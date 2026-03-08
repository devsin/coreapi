package contact

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	dbgen "github.com/devsin/coreapi/gen/db"

	"go.uber.org/zap"
)

var (
	ErrNameRequired    = errors.New("name is required")
	ErrEmailRequired   = errors.New("valid email is required")
	ErrMessageRequired = errors.New("message is required")
	ErrNameTooLong     = errors.New("name is too long")
	ErrEmailTooLong    = errors.New("email is too long")
	ErrMessageTooLong  = errors.New("message is too long")
	ErrInvalidSubject  = errors.New("invalid subject")
)

// Service handles contact-message business logic.
type Service struct {
	repo              *Repository
	log               *zap.Logger
	discordWebhookURL string
}

func NewService(repo *Repository, log *zap.Logger, discordWebhookURL string) *Service {
	return &Service{repo: repo, log: log, discordWebhookURL: discordWebhookURL}
}

func (s *Service) SubmitContactMessage(ctx context.Context, req CreateContactMessageRequest) (*MessageDTO, error) {
	// Trim
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Subject = strings.TrimSpace(req.Subject)
	req.Message = strings.TrimSpace(req.Message)

	// Validate
	if req.Name == "" {
		return nil, ErrNameRequired
	}
	if len(req.Name) > maxNameLength {
		return nil, ErrNameTooLong
	}

	if req.Email == "" {
		return nil, ErrEmailRequired
	}
	if len(req.Email) > maxEmailLength {
		return nil, ErrEmailTooLong
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, ErrEmailRequired
	}

	if req.Message == "" {
		return nil, ErrMessageRequired
	}
	if len(req.Message) > maxMessageLength {
		return nil, ErrMessageTooLong
	}

	if req.Subject == "" {
		req.Subject = "general"
	}
	if !validSubjects[req.Subject] {
		return nil, ErrInvalidSubject
	}

	msg, err := s.repo.Create(ctx, dbgen.CreateContactMessageParams{
		Name:    req.Name,
		Email:   req.Email,
		Subject: req.Subject,
		Message: req.Message,
	})
	if err != nil {
		s.log.Error("failed to save contact message", zap.Error(err))
		return nil, err
	}

	dto := &MessageDTO{
		ID:        msg.ID.String(),
		Name:      msg.Name,
		Email:     msg.Email,
		Subject:   msg.Subject,
		Message:   msg.Message,
		Status:    msg.Status,
		CreatedAt: msg.CreatedAt,
	}

	s.log.Info("contact message received",
		zap.String("id", dto.ID),
		zap.String("email", dto.Email),
		zap.String("subject", dto.Subject),
	)

	// Fire-and-forget Discord notification
	if s.discordWebhookURL != "" {
		go s.notifyDiscord(dto) //nolint:contextcheck // uses background context internally, no caller context needed
	}

	return dto, nil
}

// notifyDiscord sends a rich embed to the configured Discord webhook.
// Uses a fresh context intentionally — runs as fire-and-forget after the HTTP response.
//

func (s *Service) notifyDiscord(msg *MessageDTO) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	subjectLabels := map[string]string{
		"general":  "General Inquiry",
		"feedback": "Feedback & Suggestions",
		"bug":      "Bug Report",
		"other":    "Other",
	}
	subjectLabel := subjectLabels[msg.Subject]
	if subjectLabel == "" {
		subjectLabel = msg.Subject
	}

	// Truncate message for embed (Discord limit is 4096 for description)
	description := msg.Message
	if len(description) > 2000 {
		description = description[:2000] + "..."
	}

	payload := map[string]any{
		"embeds": []map[string]any{
			{
				"title":       fmt.Sprintf("New Contact Message: %s", subjectLabel),
				"description": description,
				"color":       0x6366F1, // indigo-500
				"fields": []map[string]any{
					{"name": "Name", "value": msg.Name, "inline": true},
					{"name": "Email", "value": msg.Email, "inline": true},
					{"name": "Subject", "value": subjectLabel, "inline": true},
				},
				"timestamp": msg.CreatedAt.Format(time.RFC3339),
				"footer": map[string]any{
					"text": fmt.Sprintf("ID: %s", msg.ID),
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		s.log.Error("failed to marshal discord payload", zap.Error(err))
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.discordWebhookURL, bytes.NewReader(body))
	if err != nil {
		s.log.Error("failed to create discord request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req) //nolint:gosec // webhook URL from server config, not user input
	if err != nil {
		s.log.Error("failed to send discord notification", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		s.log.Warn("discord webhook returned non-2xx",
			zap.Int("status", resp.StatusCode),
			zap.String("message_id", msg.ID),
		)
	}
}
