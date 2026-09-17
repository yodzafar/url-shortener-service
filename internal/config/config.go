package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	App   App
	HTTP  HTTP
	DB    DB
	Redis Redis
	JWT   JWT
}

type App struct {
	Env      string `env:"APP_ENV" envDefault:"local"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

type HTTP struct {
	Port         int           `env:"HTTP_PORT" envDefault:"8080"`
	ReadTimeout  time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"5s"`
	WriteTimeout time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"10s"`
}

type DB struct {
	URL      string `env:"DB_URL,required"`
	MaxConns int32  `env:"DB_MAX_CONNS" envDefault:"10"`
}

type Redis struct {
	Addr string `env:"REDIS_ADDR" envDefault:"localhost:6378"`
}

type JWT struct {
	Secret     string        `env:"JWT_SECRET,required"`
	AccessTTL  time.Duration `env:"JWT_ACCESS_TTL" envDefault:"15m"`
	RefreshTTL time.Duration `env:"JWT_REFRESH_TTL" envDefault:"720h"`
	Issuer     string        `env:"JWT_ISSUER" envDefault:"url-shortener"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg, err := env.ParseAs[Config]()

	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if len(cfg.JWT.Secret) < 32 {
		return nil, fmt.Errorf("config: JWT_SECRET must be at least 32 chars")
	}

	return &cfg, nil
}

func (c *Config) IsLocal() bool { return c.App.Env == "local" }
