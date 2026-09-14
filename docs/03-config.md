# 03 — Config

Config faqat **env o'zgaruvchilardan** o'qiladi (12-factor). `.env` fayli faqat local uchun.

## Kutubxonalar

- `github.com/caarlos0/env/v11` — env → struct (tavsiya, oddiy)
- `github.com/joho/godotenv` — `.env` faylni yuklash
- Alternativa: `spf13/viper` (yaml/json/env, og'irroq)

## `.env.example`

```env
APP_ENV=local
LOG_LEVEL=debug

HTTP_PORT=8080
HTTP_READ_TIMEOUT=5s
HTTP_WRITE_TIMEOUT=10s

DB_URL=postgres://postgres:postgres@localhost:5432/myservice?sslmode=disable
DB_MAX_CONNS=10

# goose CLI uchun (make migrate-*)
GOOSE_DRIVER=postgres
GOOSE_DBSTRING=postgres://postgres:postgres@localhost:5432/myservice?sslmode=disable
GOOSE_MIGRATION_DIR=migrations

REDIS_ADDR=localhost:6379

JWT_SECRET=change-me-min-32-chars-long-secret
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h
```

`.env` → `.gitignore`; `.env.example` → commit.

## `internal/config/config.go`

```go
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
    Addr string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
}

type JWT struct {
    Secret     string        `env:"JWT_SECRET,required"`
    AccessTTL  time.Duration `env:"JWT_ACCESS_TTL" envDefault:"15m"`
    RefreshTTL time.Duration `env:"JWT_REFRESH_TTL" envDefault:"720h"`
}

func Load() (*Config, error) {
    _ = godotenv.Load() // fayl bo'lmasa xato emas — prod'da real env

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
```

## Qoidalar

- `required` — bo'lmasa ishga tushmasin (fail fast). Sirlar uchun har doim.
- `config.Config`ni **faqat `app`/`main`** o'qiydi. Service/repository'ga kerak qiymatni konstruktor orqali bering (`NewTokenManager(cfg.JWT.Secret, cfg.JWT.AccessTTL)`). Test oson bo'ladi.
- Duration — `time.Duration` (`15m`, `720h`), int emas.
- Sirlarni log qilmang. `Config`ga `String()` metodi yozib, sirni `***` qiling.
