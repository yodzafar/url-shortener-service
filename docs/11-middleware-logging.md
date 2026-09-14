# 11 — Middleware, Logging, Graceful shutdown

## Middleware nima

`func(http.Handler) http.Handler` — request'ni handler'dan oldin/keyin ushlaydi. Zanjir: `RequestID → Logger → Recoverer → Auth → handler`.

```go
func Example(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // oldin
        next.ServeHTTP(w, r)
        // keyin
    })
}
```

## Logger — `pkg/logger/logger.go` (slog, std lib)

```go
package logger

import (
    "log/slog"
    "os"
)

func New(level string, isLocal bool) *slog.Logger {
    var lvl slog.Level
    _ = lvl.UnmarshalText([]byte(level)) // "debug" | "info" | "warn" | "error"

    opts := &slog.HandlerOptions{Level: lvl}
    var h slog.Handler
    if isLocal {
        h = slog.NewTextHandler(os.Stdout, opts) // o'qish oson
    } else {
        h = slog.NewJSONHandler(os.Stdout, opts) // Loki/ELK uchun
    }
    l := slog.New(h)
    slog.SetDefault(l)
    return l
}
```

Ishlatish: `log.InfoContext(ctx, "product created", "id", p.ID, "owner", p.OwnerID)` — key-value, string birlashtirish emas.

## Request logger — `internal/transport/http/middleware/logger.go`

```go
package middleware

import (
    "log/slog"
    "net/http"
    "time"

    chimw "github.com/go-chi/chi/v5/middleware"
)

func Logger(log *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor) // status kodni ushlash uchun

            next.ServeHTTP(ww, r)

            log.InfoContext(r.Context(), "http request",
                "method", r.Method,
                "path", r.URL.Path,
                "status", ww.Status(),
                "bytes", ww.BytesWritten(),
                "duration_ms", time.Since(start).Milliseconds(),
                "request_id", chimw.GetReqID(r.Context()),
                "ip", r.RemoteAddr,
            )
        })
    }
}
```

## Request ID (chi'da tayyor)

`chimw.RequestID` — `X-Request-Id` header'ni o'qiydi yoki yaratadi, context'ga qo'yadi. Log'da har doim chiqaring — bitta request'ni barcha loglarda kuzatasiz. Javob header'iga ham qaytaring:

```go
func ReturnRequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Request-Id", chimw.GetReqID(r.Context()))
        next.ServeHTTP(w, r)
    })
}
```

## Recover (panic → 500)

`chimw.Recoverer` — panic'ni ushlab 500 qaytaradi, stack'ni log qiladi. Bo'lmasa bitta panic butun serverni o'ldiradi. **Logger'dan keyin, handler'dan oldin** qo'ying.

## Context orqali qiymat uzatish (user, request id)

```go
type ctxKey int

const userKey ctxKey = iota // unexported tip — boshqa paket to'qnasha olmaydi

func WithUser(ctx context.Context, u *domain.User) context.Context {
    return context.WithValue(ctx, userKey, u)
}

func UserFromContext(ctx context.Context) *domain.User {
    u, _ := ctx.Value(userKey).(*domain.User)
    return u // nil = login qilmagan
}
```

To'liq kod `middleware/auth.go`da — [12-auth-jwt-roles.md](12-auth-jwt-roles.md). Context'ga faqat request-scoped ma'lumot (user, request id, trace). Service, DB — yo'q.

## Rate limiting

```go
import "github.com/go-chi/httprate"

r.Use(httprate.LimitByIP(100, time.Minute)) // 100 req/min per IP
```

## Graceful shutdown — `internal/app/app.go`

```go
package app

import (
    "context"
    "errors"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
    server *http.Server
    pool   *pgxpool.Pool
    log    *slog.Logger
}

func NewApp(server *http.Server, pool *pgxpool.Pool, log *slog.Logger) *App {
    return &App{server: server, pool: pool, log: log}
}

func (a *App) Run() error {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    errCh := make(chan error, 1)
    go func() {
        a.log.Info("server starting", "addr", a.server.Addr)
        if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            errCh <- err
        }
    }()

    select {
    case err := <-errCh:
        return err
    case <-ctx.Done():
        a.log.Info("shutdown signal received")
    }

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := a.server.Shutdown(shutdownCtx); err != nil { // yangi request qabul qilmaydi, eskilarini tugatadi
        return err
    }
    a.pool.Close()
    a.log.Info("server stopped")
    return nil
}
```

`cmd/api/main.go`:

```go
func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }
    app, cleanup, err := app.InitApp(context.Background(), cfg) // wire
    if err != nil {
        log.Fatal(err)
    }
    defer cleanup()
    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

Docker/Kubernetes `SIGTERM` yuboradi → server 10 soniya ichida joriy request'larni tugatib chiqadi.

## Middleware tartibi (chi)

```
RequestID → RealIP → Logger → Recoverer → Timeout → CORS → [Auth] → [RequireRole] → handler
```

- `Recoverer` Logger'dan keyin — panic ham log'ga tushsin.
- `Auth` faqat kerakli group'da, global emas.
