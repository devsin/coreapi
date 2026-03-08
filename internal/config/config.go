package config

import (
	"fmt"

	"github.com/devsin/coreapi/common/config"
)

type Config struct {
	Name    string  `env:"NAME" env-required:"true"`
	Env     string  `env:"ENV"  env-required:"true"`
	Port    int     `env:"PORT" env-required:"true"`
	DB      DB      `env-prefix:"DB_"  env-required:"true"`
	JWT     JWT     `env-prefix:"JWT_"  env-required:"true"`
	Log     Log     `env-prefix:"LOG_"  env-required:"true"`
	CORS    CORS    `env-prefix:"CORS_" env-required:"true"`
	GeoIP   GeoIP   `env-prefix:"GEOIP_"`
	Discord Discord `env-prefix:"DISCORD_"`
	Storage Storage `env-prefix:"STORAGE_"`
}

type Storage struct {
	AccountID       string `env:"ACCOUNT_ID"`
	AccessKeyID     string `env:"ACCESS_KEY_ID"`
	AccessKeySecret string `env:"ACCESS_KEY_SECRET"`
	Bucket          string `env:"BUCKET" env-default:"core-dev-uploads"`
	PublicURL       string `env:"PUBLIC_URL"` // CDN or public R2 URL prefix for serving files
}

type Discord struct {
	ContactWebhookURL string `env:"CONTACT_WEBHOOK_URL"`
}

type GeoIP struct {
	DBPath string `env:"DB_PATH" env-default:"db/geocity/GeoLite2-City.mmdb"`
}

type DB struct {
	Host     string `env:"HOST"     env-required:"true"`
	Name     string `env:"NAME"     env-required:"true"`
	Password string `env:"PASSWORD" env-required:"true"` //nolint:gosec // DB password from env, not a hardcoded secret
	Port     int    `env:"PORT"     env-required:"true"`
	SSL      string `env:"SSL"      env-required:"true"`
	Username string `env:"USERNAME" env-required:"true"`
}

func (db DB) URL() string {
	return "postgres://" + db.Username + ":" + db.Password + "@" + db.Host + ":" + fmt.Sprintf("%d", db.Port) + "/" + db.Name + "?sslmode=" + db.SSL
}

type Log struct {
	Format string `env:"FORMAT" env-required:"true"`
	Level  string `env:"LEVEL"  env-required:"true"`
}

type JWT struct {
	JWKSURL string `env:"JWKS_URL" env-required:"true"`
}

type CORS struct {
	AllowedOrigins []string `env:"ALLOWED_ORIGINS" env-required:"true"`
}

func New() Config {
	cfg, err := config.New[Config]()
	if err != nil {
		panic(err)
	}
	return *cfg
}
