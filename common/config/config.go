package config

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

func New[T any](envPath ...string) (*T, error) {
	var cfg T

	path := ".env"
	if len(envPath) > 0 && envPath[0] != "" {
		path = envPath[0]
	}

	if _, err := os.Stat(path); err == nil {
		if err := cleanenv.ReadConfig(path, &cfg); err != nil {
			return nil, err
		}
	} else {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}
