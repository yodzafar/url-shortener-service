# 04 — Database (Postgres + pgx)

## Variantlar

| Variant | Qachon |
|---------|--------|
| `jackc/pgx/v5` (pool) | Postgres, tez, to'g'ridan-to'g'ri — **tavsiya** |
| `database/sql` + `pgx/stdlib` | bir nechta DB driverga tayyor bo'lish kerak bo'lsa |
| `sqlc` | SQL yozasiz → Go kod generatsiya; repository kodini kamaytiradi |
| `GORM` | ORM; o'rganish uchun avval xom SQL tavsiya |

## `pkg/postgres/postgres.go` — pool

```go
package postgres

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
    URL      string
    MaxConns int32
}

func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
    pcfg, err := pgxpool.ParseConfig(cfg.URL)
    if err != nil {
        return nil, fmt.Errorf("postgres: parse config: %w", err)
    }
    pcfg.MaxConns = cfg.MaxConns
    pcfg.MaxConnLifetime = time.Hour
    pcfg.MaxConnIdleTime = 30 * time.Minute
    pcfg.HealthCheckPeriod = time.Minute

    pool, err := pgxpool.NewWithConfig(ctx, pcfg)
    if err != nil {
        return nil, fmt.Errorf("postgres: new pool: %w", err)
    }

    pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    if err := pool.Ping(pingCtx); err != nil {
        pool.Close()
        return nil, fmt.Errorf("postgres: ping: %w", err)
    }
    return pool, nil
}
```

`app`da: `defer pool.Close()`.

## Migratsiyalar (golang-migrate)

Fayl: `NNNNNN_nomi.up.sql` / `.down.sql`.

`migrations/000001_create_users.up.sql`:

```sql
CREATE TABLE users (
    id            BIGSERIAL    PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT         NOT NULL,
    role          VARCHAR(32)  NOT NULL DEFAULT 'user',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
```

`migrations/000002_create_products.up.sql`:

```sql
CREATE TABLE products (
    id          BIGSERIAL     PRIMARY KEY,
    owner_id    BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255)  NOT NULL,
    price       NUMERIC(12,2) NOT NULL CHECK (price >= 0),
    status      VARCHAR(32)   NOT NULL DEFAULT 'draft',
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE INDEX idx_products_owner_id ON products (owner_id);
```

`.down.sql` — teskarisi: `DROP TABLE IF EXISTS products;`

```bash
migrate create -ext sql -dir migrations -seq create_users
migrate -path migrations -database "$DB_URL" up
migrate -path migrations -database "$DB_URL" down 1
migrate -path migrations -database "$DB_URL" version
migrate -path migrations -database "$DB_URL" force 1   # dirty holatni tuzatish
```

### Migratsiyani binary ichida ishga tushirish (ixtiyoriy)

```go
//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(dbURL string) error {
    src, err := iofs.New(migrationsFS, "migrations")
    if err != nil { return err }
    m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
    if err != nil { return err }
    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        return err
    }
    return nil
}
```

## pgx asoslari

```go
// bitta qator
err := pool.QueryRow(ctx, `SELECT id, email FROM users WHERE id = $1`, id).Scan(&u.ID, &u.Email)
if errors.Is(err, pgx.ErrNoRows) { /* not found */ }

// ko'p qator → struct slice (pgx v5)
rows, err := pool.Query(ctx, `SELECT id, name, price FROM products LIMIT $1`, limit)
items, err := pgx.CollectRows(rows, pgx.RowToStructByName[productRow])   // rows.Close() o'zi qiladi

// bitta qator → struct
row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[productRow])

// exec
tag, err := pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
if tag.RowsAffected() == 0 { /* not found */ }
```

Har doim `$1, $2` placeholder — string birlashtirish YO'Q (SQL injection).

## Tranzaksiya

`internal/repository/postgres/tx.go`:

```go
package postgres

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"
)

// DB — *pgxpool.Pool ham, pgx.Tx ham shu interfeysni qanoatlantiradi.
// Repository shu interfeys orqali ishlasa — tx ichida ham, tashqarida ham ishlaydi.
type DB interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) (err error) {
    tx, err := pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer func() {
        if p := recover(); p != nil {
            _ = tx.Rollback(ctx)
            panic(p)
        }
        if err != nil {
            _ = tx.Rollback(ctx)
            return
        }
        err = tx.Commit(ctx)
    }()
    return fn(tx)
}
```

Service tranzaksiyani bilmasligi kerak. Ilg'or yo'l: `service/ports.go`da `TxManager interface { Do(ctx, func(ctx context.Context) error) error }` — tx context ichida uzatiladi, repository context'dan tx'ni oladi (bo'lmasa pool). Boshida bir nechta query bitta repository metodida bo'lsa, tx'ni repository ichida oching.

## Postgres xato kodlari

```go
var pgErr *pgconn.PgError
if errors.As(err, &pgErr) {
    switch pgErr.Code {
    case "23505": return domain.ErrEmailTaken   // unique_violation
    case "23503": return domain.ErrRelatedNotFound // foreign_key_violation
    }
}
```
