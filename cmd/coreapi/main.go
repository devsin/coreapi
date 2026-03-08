package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/devsin/coreapi/common/db"
	"github.com/devsin/coreapi/common/httpx"
	"github.com/devsin/coreapi/common/logger"
	"github.com/devsin/coreapi/internal/app"
	"github.com/devsin/coreapi/internal/config"
	"github.com/devsin/coreapi/internal/contact"
	"github.com/devsin/coreapi/internal/insights"
	"github.com/devsin/coreapi/internal/media"
	"github.com/devsin/coreapi/internal/notification"
	"github.com/devsin/coreapi/internal/search"
	"github.com/devsin/coreapi/internal/session"
	"github.com/devsin/coreapi/internal/settings"
	"github.com/devsin/coreapi/internal/storage"
	"github.com/devsin/coreapi/internal/users"

	"go.uber.org/zap"
)

// userNotifAdapter wraps notification.Service to satisfy users.Notifier.
type userNotifAdapter struct {
	svc *notification.Service
}

func (a *userNotifAdapter) Notify(ctx context.Context, p users.NotifyParams) {
	a.svc.Notify(ctx, notification.CreateParams(p))
}

func main() {
	ctx := context.Background()

	cfg := config.New()

	logg, err := logger.New(cfg.Env, cfg.Log.Level)
	if err != nil {
		log.Fatalf("logger init failed: %v", err)
	}
	defer func() { _ = logg.Sync() }() //nolint:errcheck // best-effort flush on shutdown

	pool, err := db.OpenPool(ctx, cfg.DB.URL())
	if err != nil {
		logg.Fatal("database init failed", zap.Error(err))
	}
	defer pool.Close()

	userRepo := users.NewRepository(pool)
	userSvc := users.NewService(userRepo, logg)
	userHandler := users.NewHandler(userSvc, logg)

	settingsRepo := settings.NewRepository(pool)
	settingsSvc := settings.NewService(settingsRepo, logg)
	settingsHandler := settings.NewHandler(settingsSvc, logg)

	geoResolver, err := insights.NewGeoIPResolver(cfg.GeoIP.DBPath)
	if err != nil {
		logg.Fatal("geoip init failed", zap.Error(err))
	}
	defer geoResolver.Close()

	insightsRepo := insights.NewRepository(pool)
	dailyInsightsRepo := insights.NewDailyInsightsRepo(pool)
	insightsSvc := insights.NewService(insightsRepo, dailyInsightsRepo, geoResolver, logg)
	insightsHandler := insights.NewHandler(insightsSvc, logg)

	sessionRepo := session.NewRepository(pool)
	sessionSvc := session.NewService(sessionRepo, logg)
	sessionHandler := session.NewHandler(sessionSvc, logg)

	searchSvc := search.NewService(pool, logg)
	searchHandler := search.NewHandler(searchSvc, logg)

	contactRepo := contact.NewRepository(pool)
	contactSvc := contact.NewService(contactRepo, logg, cfg.Discord.ContactWebhookURL)
	contactHandler := contact.NewHandler(contactSvc, logg)

	notifRepo := notification.NewRepository(pool)
	notifHub := notification.NewHub()
	notifSvc := notification.NewService(notifRepo, userRepo, notifHub, logg)
	notifHandler := notification.NewHandler(notifSvc, notifHub, logg)

	// Media / upload service (Cloudflare R2)
	r2Store := storage.NewR2Store(storage.R2Config{
		AccountID:       cfg.Storage.AccountID,
		AccessKeyID:     cfg.Storage.AccessKeyID,
		AccessKeySecret: cfg.Storage.AccessKeySecret,
		Bucket:          cfg.Storage.Bucket,
		PublicURL:       cfg.Storage.PublicURL,
	})
	mediaSvc := media.NewService(r2Store, logg)
	mediaHandler := media.NewHandler(mediaSvc, logg)

	// Wire profile privacy checks into user service
	userSvc.SetPrivacyChecker(settingsSvc)

	// Wire notifications into user service
	userSvc.SetNotifier(&userNotifAdapter{svc: notifSvc})

	// Rate limiter for public GET endpoints: 300 req/min per IP, burst of 40.
	publicLimiter := httpx.NewRateLimiter(300, time.Minute, 40)
	defer publicLimiter.Stop()

	// Rate limiter for public tracking endpoints: 30 req/min per IP, burst of 10.
	trackingLimiter := httpx.NewRateLimiter(30, time.Minute, 10)
	defer trackingLimiter.Stop()

	// Backfill daily_insights from raw events (last 90 days, idempotent).
	go func() {
		now := time.Now().UTC()
		from := now.AddDate(0, 0, -90)
		if err := insightsSvc.BackfillDailyInsights(context.Background(), from, now); err != nil {
			logg.Error("daily insights backfill failed", zap.Error(err))
		}
	}()

	// Cleanup stale sessions (>30 days) on startup.
	go func() {
		if err := sessionSvc.CleanupStaleSessions(context.Background()); err != nil {
			logg.Error("stale session cleanup failed", zap.Error(err))
		}
	}()

	// Periodically clean up the in-memory dedup cache.
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			sessionSvc.CleanupDedupCache()
		}
	}()

	router := app.NewRouter(app.RouterDeps{
		Config: cfg,
		Log:    logg,
		Limiters: struct {
			Public   *httpx.RateLimiter
			Tracking *httpx.RateLimiter
		}{
			Public:   publicLimiter,
			Tracking: trackingLimiter,
		},
		Handlers: struct {
			User         *users.Handler
			Settings     *settings.Handler
			Insights     *insights.Handler
			Session      *session.Handler
			Search       *search.Handler
			Contact      *contact.Handler
			Notification *notification.Handler
			Media        *media.Handler
		}{
			User:         userHandler,
			Settings:     settingsHandler,
			Insights:     insightsHandler,
			Session:      sessionHandler,
			Search:       searchHandler,
			Contact:      contactHandler,
			Notification: notifHandler,
			Media:        mediaHandler,
		},
		SessionSvc:  sessionSvc,
		GeoResolver: geoResolver,
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	if err := httpx.Run(ctx, addr, router, logg); err != nil {
		logg.Fatal("server error", zap.Error(err))
	}
}
