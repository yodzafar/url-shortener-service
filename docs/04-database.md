# 04 — Database (Postgres + pgx)

## Variantlar

| Variant | Qachon |
|---------|--------|
| `jackc/pgx/v5` (pool) | Postgres, tez, to'g'ridan-to'g'ri — **tavsiya** |
| `database/sql` + `pgx/stdlib` | bir nechta DB driverga tayyor bo'lish kerak bo'lsa |
| `sqlc` | SQL yozasiz → Go kod generatsiya; repository kodini kamaytiradi — [23-sqlc.md](23-sqlc.md) |
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

## Migratsiyalar (goose)

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
go get github.com/pressly/goose/v3          # binary ichida ishga tushirish uchun
```

Bitta faylda `Up` va `Down`. Fayl nomi: `<timestamp>_nomi.sql` (`goose create` o'zi beradi).

```bash
goose -dir migrations create create_users sql        # migrations/20260914120000_create_users.sql
goose -dir migrations create create_products sql
```

`migrations/20260914120000_create_users.sql`:

```sql
-- +goose Up
CREATE TABLE users (
    id            BIGSERIAL    PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT         NOT NULL,
    role          VARCHAR(32)  NOT NULL DEFAULT 'user',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS users;
```

`migrations/20260914120100_create_products.sql`:

```sql
-- +goose Up
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

-- +goose Down
DROP TABLE IF EXISTS products;
```

Funksiya/trigger kabi `;` ichida `;` bo'lgan statement uchun:

```sql
-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
```

`CREATE INDEX CONCURRENTLY` tranzaksiyada ishlamaydi — fayl boshiga `-- +goose NO TRANSACTION`.

### Buyruqlar

Env orqali (`.env`ga qo'shing, Makefile `export` qiladi):

```env
GOOSE_DRIVER=postgres
GOOSE_DBSTRING=postgres://postgres:postgres@localhost:5432/myservice?sslmode=disable
GOOSE_MIGRATION_DIR=migrations
```

```bash
goose up              # hammasini qo'llash
goose up-by-one       # bittasini
goose down            # oxirgisini qaytarish
goose status          # qaysilari qo'llangan — jadval
goose version         # joriy versiya
goose redo            # down + up (oxirgisini qayta tekshirish)
goose reset           # hammasini qaytarish (faqat lokal!)
goose validate        # fayllar sintaksisi
```

Env'siz: `goose -dir migrations postgres "$DB_URL" up`.

Ketma-ket raqam (`00001_`) xohlasangiz: `goose create -s ...`. Jamoada timestamp xavfsizroq (to'qnashmaydi).

### Binary ichida ishga tushirish (ixtiyoriy)

goose `*sql.DB` bilan ishlaydi — `pgx/stdlib` driver orqali:

```go
package migrations

import (
    "database/sql"
    "embed"
    "fmt"

    _ "github.com/jackc/pgx/v5/stdlib" // "pgx" driver database/sql uchun
    "github.com/pressly/goose/v3"
)

//go:embed *.sql
var FS embed.FS

func Up(dbURL string) error {
    db, err := sql.Open("pgx", dbURL)
    if err != nil {
        return fmt.Errorf("migrations: open: %w", err)
    }
    defer db.Close()

    goose.SetBaseFS(FS)
    if err := goose.SetDialect("postgres"); err != nil {
        return err
    }
    if err := goose.Up(db, "."); err != nil {
        return fmt.Errorf("migrations: up: %w", err)
    }
    return nil
}
```

Fayl `migrations/migrate.go` bo'lsa `//go:embed *.sql` va `goose.Up(db, ".")`. Startup'da chaqirish: `if cfg.IsLocal() { migrations.Up(cfg.DB.URL) }`. Prod'da alohida qadam (CI/CD) tavsiya.

**Best practice — bitta pool.** `sql.Open` yangi ulanishlar to'plamini ochadi. Mavjud `*pgxpool.Pool` ustida `*sql.DB` yasash mumkin, shunda goose ham, repository'lar ham bitta pool'ni ishlatadi:

```go
import "github.com/jackc/pgx/v5/stdlib"

func Up(pool *pgxpool.Pool) error {
    db := stdlib.OpenDBFromPool(pool) // yangi ulanish ochmaydi
    defer db.Close()                  // faqat wrapper yopiladi, pool emas
    goose.SetBaseFS(FS)
    if err := goose.SetDialect("postgres"); err != nil {
        return err
    }
    return goose.Up(db, ".")
}
```

Umumiy qoida: **`pgxpool` asosiy**, `database/sql` faqat uni talab qiladigan kutubxonalar (goose, sqlx, sqlmock) uchun ko'prik. Ikki alohida pool ochmang.

### Go'da migratsiya (ma'lumot transformatsiyasi kerak bo'lsa)

```go
// migrations/20260914130000_backfill_roles.go
package migrations

func init() {
    goose.AddMigrationContext(upBackfillRoles, downBackfillRoles)
}

func upBackfillRoles(ctx context.Context, tx *sql.Tx) error {
    _, err := tx.ExecContext(ctx, `UPDATE users SET role = 'user' WHERE role = ''`)
    return err
}

func downBackfillRoles(ctx context.Context, tx *sql.Tx) error { return nil }
```

Go migratsiyalar faqat binary ichidan (`goose.Up`) ishlaydi, CLI ularni ko'rmaydi.

### Alternativa: golang-migrate

Ikki fayl (`.up.sql`/`.down.sql`), ko'p DB (Mongo, Cassandra), tashqi manbalar (S3). SQL DB uchun goose soddaroq. Taqqoslash: `goose status` jadval beradi, "dirty" holat yo'q, Go migratsiya bor.

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

## Struct'ga o'qish (struct scan) — pgx v5

`pgx.CollectRows` / `pgx.CollectOneRow` + row mapper:

| Mapper | Ustunni field'ga bog'lash | Qachon |
|--------|---------------------------|--------|
| `RowToStructByName[T]` | ustun nomi ↔ `db:"..."` teg | **tavsiya** — tartib muhim emas |
| `RowToStructByPos[T]` | ustun tartibi ↔ field tartibi | tez, lekin `SELECT` tartibi o'zgarsa buziladi |
| `RowToStructByNameLax[T]` | ByName, struct'da ortiqcha field bo'lsa xato bermaydi | qisman `SELECT` |
| `RowTo[T]` | bitta ustun → oddiy tip | `SELECT id FROM ...`, `count(*)` |
| `RowToAddrOfStructByName[T]` | ByName, `[]*T` qaytaradi | pointer slice kerak bo'lsa |

```go
type productRow struct {
    ID        int64      `db:"id"`
    OwnerID   int64      `db:"owner_id"`
    Name      string     `db:"name"`
    Price     float64    `db:"price"`
    ExpiresAt *time.Time `db:"expires_at"` // NULL bo'lishi mumkin → pointer
    CreatedAt time.Time  `db:"created_at"`
}

// Ko'p qator → []productRow (rows.Close() ni CollectRows o'zi chaqiradi)
rows, err := pool.Query(ctx, `SELECT id, owner_id, name, price, expires_at, created_at FROM products`)
if err != nil { return nil, err }
items, err := pgx.CollectRows(rows, pgx.RowToStructByName[productRow])

// Bitta qator → productRow (Query ishlatiladi, QueryRow emas)
rows, err = pool.Query(ctx, `SELECT ... FROM products WHERE id = $1`, id)
if err != nil { return nil, err }
item, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[productRow])
if errors.Is(err, pgx.ErrNoRows) { return nil, domain.ErrProductNotFound }

// Bitta ustun → []int64
rows, _ = pool.Query(ctx, `SELECT id FROM products WHERE owner_id = $1`, ownerID)
ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])

// count(*)
rows, _ = pool.Query(ctx, `SELECT count(*) FROM products`)
total, err := pgx.CollectOneRow(rows, pgx.RowTo[int])

// JOIN natijasi — embedded struct
type productWithOwner struct {
    productRow
    OwnerEmail string `db:"owner_email"`
}
```

Qoidalar:
- Ustun soni = field soni (`ByName`/`ByPos`), aks holda `number of field descriptions must equal number of destinations`. Qisman `SELECT` → `Lax`.
- `SELECT *` yozmang: ustun qo'shilsa `ByName` xato beradi, `ByPos` noto'g'ri qiymat o'qiydi.
- NULL ustun → pointer (`*time.Time`, `*string`) yoki `pgtype.Text`. Oddiy `string`ga NULL scan — xato.
- Teg nomi `db` (`json` emas). Teg bo'lmasa field nomi case-insensitive moslanadi (`OwnerID` ↔ `ownerid`, `owner_id` emas) — shuning uchun teg yozing.
- Field'ni o'tkazib yuborish: `db:"-"`.
- 2–3 ustun uchun qo'lda `QueryRow(...).Scan(&a, &b)` ham yetarli; 10+ ustunda `CollectRows` xavfsizroq.

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
