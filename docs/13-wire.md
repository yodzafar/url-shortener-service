# 13 — Dependency Injection: Google Wire

## Nima uchun DI

`main.go`da qo'lda:

```go
pool := postgres.NewPool(...)
userRepo := pgrepo.NewUserRepository(pool)
productRepo := pgrepo.NewProductRepository(pool)
hasher := hash.NewBcrypt()
tokens := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.AccessTTL, "myservice")
authSvc := service.NewAuthService(userRepo, hasher, tokens, refreshStore, cfg.JWT.RefreshTTL, clock.Real{})
productSvc := service.NewProductService(productRepo, clock.Real{}, log)
authH := handler.NewAuthHandler(authSvc, v)
...
```

20 ta bog'liqlikda bu 60 qator va tartib xatosiga to'la. **Wire** — konstruktorlarni beradi, u kompilyatsiya vaqtida shu kodni **generatsiya qiladi**. Reflection yo'q, runtime magic yo'q — oddiy Go kodi chiqadi.

```bash
go get github.com/google/wire
go install github.com/google/wire/cmd/wire@latest
```

## Provider = konstruktor

Wire uchun har bir konstruktor "provider": kirish = bog'liqliklar, chiqish = tip (+ ixtiyoriy `error`, `cleanup func()`).

```go
func NewProductService(repo ProductRepository, clock Clock, log *slog.Logger) *ProductService
//                     ↑ Wire shu tiplarni boshqa providerlardan topadi
```

## `internal/app/wire.go` (qo'lda yoziladi)

```go
//go:build wireinject
// +build wireinject

package app

import (
    "context"

    "github.com/google/wire"

    "github.com/yodzafar/myservice/internal/config"
    "github.com/yodzafar/myservice/internal/repository/postgres"
    "github.com/yodzafar/myservice/internal/service"
    httptransport "github.com/yodzafar/myservice/internal/transport/http"
    "github.com/yodzafar/myservice/internal/transport/http/handler"
    "github.com/yodzafar/myservice/internal/transport/http/middleware"
)

// InitApp — inject funksiya. Tanasi Wire uchun "recept"; wire_gen.go'da haqiqiy kod bo'ladi.
func InitApp(ctx context.Context, cfg *config.Config) (*App, func(), error) {
    wire.Build(
        infraSet,
        repositorySet,
        serviceSet,
        transportSet,
        NewApp,
    )
    return nil, nil, nil
}
```

## `internal/app/providers.go` (provider set'lar — wireinject tegi YO'Q)

```go
package app

import (
    "github.com/google/wire"

    "github.com/yodzafar/myservice/internal/config"
    "github.com/yodzafar/myservice/internal/repository/postgres"
    "github.com/yodzafar/myservice/internal/repository/redis"
    "github.com/yodzafar/myservice/internal/service"
    httptransport "github.com/yodzafar/myservice/internal/transport/http"
    "github.com/yodzafar/myservice/internal/transport/http/handler"
    "github.com/yodzafar/myservice/internal/transport/http/middleware"
    "github.com/yodzafar/myservice/pkg/clock"
    "github.com/yodzafar/myservice/pkg/hash"
    "github.com/yodzafar/myservice/pkg/jwt"
    "github.com/yodzafar/myservice/pkg/logger"
    pgpool "github.com/yodzafar/myservice/pkg/postgres"
    "github.com/yodzafar/myservice/pkg/validator"
)

// --- infra: logger, db, validator, hasher, jwt ---
var infraSet = wire.NewSet(
    provideLogger,
    providePool,
    provideRedis,
    validator.New,
    hash.NewBcrypt,
    provideTokenManager,
    wire.Struct(new(clock.Real)),
    // interfeys ↔ implementatsiya bog'lash
    wire.Bind(new(service.PasswordHasher), new(*hash.Bcrypt)),
    wire.Bind(new(service.TokenManager), new(*jwt.Manager)),
    wire.Bind(new(service.Clock), new(clock.Real)),
)

var repositorySet = wire.NewSet(
    postgres.NewUserRepository,
    postgres.NewProductRepository,
    redis.NewRefreshTokenStore,
    wire.Bind(new(service.UserRepository), new(*postgres.UserRepository)),
    wire.Bind(new(service.ProductRepository), new(*postgres.ProductRepository)),
    wire.Bind(new(service.RefreshTokenStore), new(*redis.RefreshTokenStore)),
)

var serviceSet = wire.NewSet(
    provideAuthService,
    service.NewProductService,
)

var transportSet = wire.NewSet(
    handler.NewAuthHandler,
    handler.NewProductHandler,
    middleware.NewAuth,
    wire.Struct(new(httptransport.RouterDeps), "*"), // barcha field'larni to'ldir
    httptransport.NewRouter,
    provideServer,
)

// --- config'dan alohida qiymat olib beradigan providerlar ---

func provideLogger(cfg *config.Config) *slog.Logger {
    return logger.New(cfg.App.LogLevel, cfg.IsLocal())
}

// cleanup qaytaradi — Wire uni InitApp'ning cleanup'iga qo'shadi
func providePool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, func(), error) {
    pool, err := pgpool.NewPool(ctx, pgpool.Config{URL: cfg.DB.URL, MaxConns: cfg.DB.MaxConns})
    if err != nil {
        return nil, nil, err
    }
    return pool, func() { pool.Close() }, nil
}

func provideRedis(cfg *config.Config) *goredis.Client {
    return goredis.NewClient(&goredis.Options{Addr: cfg.Redis.Addr})
}

func provideTokenManager(cfg *config.Config) *jwt.Manager {
    return jwt.NewManager(cfg.JWT.Secret, cfg.JWT.AccessTTL, "myservice")
}

func provideAuthService(users service.UserRepository, h service.PasswordHasher, t service.TokenManager,
    store service.RefreshTokenStore, cfg *config.Config, c service.Clock) *service.AuthService {
    return service.NewAuthService(users, h, t, store, cfg.JWT.RefreshTTL, c)
}

func provideServer(cfg *config.Config, h http.Handler) *http.Server {
    return httptransport.NewServer(cfg.HTTP.Port, h, cfg.HTTP.ReadTimeout, cfg.HTTP.WriteTimeout)
}
```

Nega `provideXxx`? Wire bir xil tipdan ikkitasini ajrata olmaydi (`string` secret va `string` addr). Shuning uchun `*config.Config`ni olib kerakli qismini beradigan kichik funksiyalar yoziladi.

`pgxpool.Pool` `postgres.DB` interfeysiga bind: `wire.Bind(new(postgres.DB), new(*pgxpool.Pool))` — `repositorySet`ga qo'shing.

## Generatsiya

```bash
cd internal/app && wire        # yoki loyiha ildizidan: wire ./internal/app
# yoki
go generate ./...              # wire.go'ga: //go:generate go run github.com/google/wire/cmd/wire
```

Natija `wire_gen.go` (commit qilinadi, qo'lda tahrirlanmaydi):

```go
// Code generated by Wire. DO NOT EDIT.
//go:build !wireinject

func InitApp(ctx context.Context, cfg *config.Config) (*App, func(), error) {
    logger := provideLogger(cfg)
    pool, cleanup, err := providePool(ctx, cfg)
    if err != nil {
        return nil, nil, err
    }
    userRepository := postgres.NewUserRepository(pool)
    ...
    app := NewApp(server, pool, logger)
    return app, func() { cleanup() }, nil
}
```

`wire.go`dagi `//go:build wireinject` va `wire_gen.go`dagi `!wireinject` — ikkalasi bir vaqtda kompilyatsiya qilinmaydi.

## Tez-tez uchraydigan xatolar

| Xato | Yechim |
|------|--------|
| `no provider found for service.ProductRepository` | `wire.Bind` unutilgan |
| `multiple bindings for *slog.Logger` | bir tip ikki marta provide qilingan |
| `unused provider set` | set ishlatilmayapti — olib tashlang |
| `inject InitApp: unused provider` | provider bor, hech kim ishlatmaydi |
| Bir tip (`string`) ikki xil ma'noda | `provideXxx` funksiya yoki `type Secret string` alias |

## Wire vs boshqalar

| Vosita | Yondashuv | Tavsiya |
|--------|-----------|---------|
| **Wire** | compile-time codegen, magic yo'q | ✅ o'rganish va prod uchun |
| `uber-go/fx` | runtime, lifecycle hooks, katta loyihalar | Uber-style katta servislar |
| `uber-go/dig` | runtime reflection | fx'ning asosi |
| Qo'lda | `main.go`da hammasi | 5–10 bog'liqlikkacha yetarli |

Wire'ni tushunish uchun avval bir marta qo'lda yig'ib ko'ring — keyin nima generatsiya qilayotganini aniq ko'rasiz.
