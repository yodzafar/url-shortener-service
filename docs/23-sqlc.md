# 23 — sqlc: SQL'dan Go kod generatsiya

`sqlc` — siz **SQL yozasiz**, u tip-xavfsiz Go funksiyalar va struct'lar generatsiya qiladi. Qo'lda `Scan`, `productRow`, `CollectRows` yozish shart emas. Xato SQL kompilyatsiya vaqtida (generatsiyada) topiladi, runtime'da emas.

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
sqlc version
```

Kutubxona kerak emas — generatsiya qilingan kod faqat `pgx/v5` ga bog'liq.

## Loyihadagi o'rni

```
internal/
├── repository/
│   └── postgres/
│       ├── sqlc/                  # GENERATSIYA (qo'lda tahrirlanmaydi)
│       │   ├── db.go              # Queries struct, DBTX interfeysi
│       │   ├── models.go          # jadval → struct (User, Product)
│       │   ├── users.sql.go       # har query → metod
│       │   └── products.sql.go
│       ├── queries/               # SIZ YOZASIZ
│       │   ├── users.sql
│       │   └── products.sql
│       ├── product_repository.go  # service.ProductRepository implementatsiyasi: sqlc → domain mapping
│       └── user_repository.go
migrations/                        # goose fayllari — sqlc schema'ni shu yerdan o'qiydi
sqlc.yaml                          # loyiha ildizi
```

Clean Architecture buzilmaydi: service hali ham `service.ProductRepository` interfeysini ko'radi. sqlc faqat repository **ichida**. Generatsiya qilingan `sqlc.Product` — DB struct (`productRow` o'rnini bosadi), `domain.Product` emas.

## `sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "migrations"                          # goose fayllarini tushunadi (Up qismini o'qiydi)
    queries: "internal/repository/postgres/queries"
    gen:
      go:
        package: "sqlc"
        out: "internal/repository/postgres/sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: false                     # DB struct'da json teg kerak emas
        emit_pointers_for_null_types: true        # NULL → *T (pgtype.Text o'rniga)
        emit_empty_slices: true                   # bo'sh natija → [] (nil emas)
        emit_methods_with_db_argument: false
        overrides:
          - db_type: "timestamptz"
            go_type: "time.Time"
          - db_type: "timestamptz"
            nullable: true
            go_type:
              type: "time.Time"
              pointer: true
          - db_type: "numeric"
            go_type: "float64"                    # o'rganish uchun; prod'da shopspring/decimal
```

## Query yozish — `queries/products.sql`

Har query ustida komment: `-- name: <FunksiyaNomi> :<natija turi>`.

| Suffiks | Qaytaradi |
|---------|-----------|
| `:one` | bitta qator (`T, error`), topilmasa `pgx.ErrNoRows` |
| `:many` | `[]T, error` |
| `:exec` | `error` |
| `:execrows` | `int64, error` (RowsAffected) |
| `:execresult` | `pgconn.CommandTag, error` |
| `:batchexec`, `:batchmany` | batch (pgx) |
| `:copyfrom` | `COPY` bilan ommaviy insert |

```sql
-- name: CreateProduct :one
INSERT INTO products (owner_id, name, price, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetProduct :one
SELECT * FROM products WHERE id = $1;

-- name: ListProducts :many
SELECT * FROM products
WHERE (sqlc.narg('owner_id')::bigint IS NULL OR owner_id = sqlc.narg('owner_id'))
  AND (sqlc.narg('status')::text  IS NULL OR status   = sqlc.narg('status'))
ORDER BY id DESC
LIMIT $1 OFFSET $2;

-- name: CountProducts :one
SELECT count(*) FROM products
WHERE (sqlc.narg('owner_id')::bigint IS NULL OR owner_id = sqlc.narg('owner_id'))
  AND (sqlc.narg('status')::text  IS NULL OR status   = sqlc.narg('status'));

-- name: UpdateProduct :execrows
UPDATE products
SET name = $2, price = $3, status = $4, updated_at = $5
WHERE id = $1;

-- name: DeleteProduct :execrows
DELETE FROM products WHERE id = $1;

-- name: GetProductWithOwner :one
SELECT sqlc.embed(products), sqlc.embed(users)
FROM products
JOIN users ON users.id = products.owner_id
WHERE products.id = $1;
```

- `sqlc.arg('name')` / `@name` — nomli parametr (`$1` o'rniga).
- `sqlc.narg('name')` — **nullable** parametr (`*T`) — ixtiyoriy filter uchun.
- `sqlc.embed(table)` — JOIN'da ikki struct'ni ichma-ich qaytaradi.
- `SELECT *` bu yerda **xavfsiz**: sqlc schema'ni biladi, ustun qo'shilsa qayta generatsiya qilasiz va kompilyator xatoni ko'rsatadi.

## `queries/users.sql`

```sql
-- name: CreateUser :one
INSERT INTO users (email, password_hash, role, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: ExistsUserByEmail :one
SELECT EXISTS (SELECT 1 FROM users WHERE email = $1);

-- name: ListUsers :many
SELECT * FROM users
WHERE (sqlc.narg('role')::text IS NULL OR role = sqlc.narg('role'))
ORDER BY id DESC
LIMIT $1 OFFSET $2;

-- name: UpdateUserPassword :execrows
UPDATE users SET password_hash = $2, updated_at = $3 WHERE id = $1;

-- name: UpdateUserRole :execrows
UPDATE users SET role = $2, updated_at = $3 WHERE id = $1;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = $1;

-- name: GetUsersByIDs :many
SELECT * FROM users WHERE id = ANY(@ids::bigint[]);
```

Repository'da unique xato tarjimasi sqlc bilan ham xuddi shunday (sqlc `pgconn.PgError`ni o'zgartirmaydi):

```go
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
    row, err := r.q.CreateUser(ctx, toCreateUserParams(u))
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            return domain.ErrEmailTaken
        }
        return fmt.Errorf("user repo: create: %w", err)
    }
    u.ID = row.ID
    return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
    row, err := r.q.GetUserByEmail(ctx, email)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, domain.ErrUserNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("user repo: get by email: %w", err)
    }
    u := toDomainUser(row)
    return &u, nil
}
```

## Generatsiya

```bash
sqlc generate      # yoki: make sqlc
sqlc vet           # query'larni tekshirish (DB'ga ulanib EXPLAIN ham qila oladi)
```

Natija (`products.sql.go`, qisqartirilgan):

```go
type Product struct {
    ID        int64
    OwnerID   int64
    Name      string
    Price     float64
    Status    string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type CreateProductParams struct {
    OwnerID   int64
    Name      string
    Price     float64
    Status    string
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (q *Queries) CreateProduct(ctx context.Context, arg CreateProductParams) (Product, error)
func (q *Queries) GetProduct(ctx context.Context, id int64) (Product, error)
func (q *Queries) ListProducts(ctx context.Context, arg ListProductsParams) ([]Product, error)
func (q *Queries) UpdateProduct(ctx context.Context, arg UpdateProductParams) (int64, error)

// db.go
type DBTX interface {
    Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
    Query(context.Context, string, ...interface{}) (pgx.Rows, error)
    QueryRow(context.Context, string, ...interface{}) pgx.Row
}
func New(db DBTX) *Queries
func (q *Queries) WithTx(tx pgx.Tx) *Queries
```

`DBTX` — `*pgxpool.Pool` ham, `pgx.Tx` ham mos keladi (04-database.md dagi `DB` interfeysi bilan bir xil g'oya).

## Repository — sqlc'ni domain'ga o'rash

`internal/repository/postgres/product_repository.go`:

```go
package postgres

import (
    "context"
    "errors"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/yodzafar/myservice/internal/domain"
    "github.com/yodzafar/myservice/internal/repository/postgres/sqlc"
    "github.com/yodzafar/myservice/internal/service"
)

var _ service.ProductRepository = (*ProductRepository)(nil)

type ProductRepository struct {
    q *sqlc.Queries
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
    return &ProductRepository{q: sqlc.New(pool)}
}

func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
    row, err := r.q.CreateProduct(ctx, sqlc.CreateProductParams{
        OwnerID: p.OwnerID, Name: p.Name, Price: p.Price, Status: string(p.Status),
        CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
    })
    if err != nil {
        return fmt.Errorf("product repo: create: %w", err)
    }
    p.ID = row.ID
    return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
    row, err := r.q.GetProduct(ctx, id)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, domain.ErrProductNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("product repo: get: %w", err)
    }
    p := toDomainProduct(row)
    return &p, nil
}

func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
    n, err := r.q.UpdateProduct(ctx, sqlc.UpdateProductParams{
        ID: p.ID, Name: p.Name, Price: p.Price, Status: string(p.Status), UpdatedAt: p.UpdatedAt,
    })
    if err != nil {
        return fmt.Errorf("product repo: update: %w", err)
    }
    if n == 0 {
        return domain.ErrProductNotFound
    }
    return nil
}

func (r *ProductRepository) List(ctx context.Context, f service.ProductFilter) ([]domain.Product, int, error) {
    var status *string
    if f.Status != nil {
        s := string(*f.Status)
        status = &s
    }
    rows, err := r.q.ListProducts(ctx, sqlc.ListProductsParams{
        OwnerID: f.OwnerID, Status: status, Limit: int32(f.Limit), Offset: int32(f.Offset),
    })
    if err != nil {
        return nil, 0, fmt.Errorf("product repo: list: %w", err)
    }
    total, err := r.q.CountProducts(ctx, sqlc.CountProductsParams{OwnerID: f.OwnerID, Status: status})
    if err != nil {
        return nil, 0, fmt.Errorf("product repo: count: %w", err)
    }
    return slices.Map(rows, toDomainProduct), int(total), nil // pkg/slices generic yordamchi
}

// mapper.go — sqlc struct → domain (06-repository.md)
func toDomainProduct(r sqlc.Product) domain.Product {
    return domain.Product{
        ID: r.ID, OwnerID: r.OwnerID, Name: r.Name, Price: r.Price,
        Status: domain.ProductStatus(r.Status), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
    }
}
```

Repository ~2 barobar qisqardi: SQL, `Scan`, `productRow` — hammasi generatsiya.

## Tranzaksiya

```go
func (r *ProductRepository) CreateWithLog(ctx context.Context, p *domain.Product) error {
    return WithTx(ctx, r.pool, func(tx pgx.Tx) error {  // 04-database.md dagi WithTx
        q := r.q.WithTx(tx)
        row, err := q.CreateProduct(ctx, ...)
        if err != nil { return err }
        return q.CreateAuditLog(ctx, ...)
    })
}
```

Buning uchun `ProductRepository` ichida `pool *pgxpool.Pool` ham saqlanadi.

## Makefile

```makefile
sqlc: ## SQL → Go
	sqlc generate

sqlc-vet: ## Query'larni tekshirish
	sqlc vet

generate: wire swagger mocks sqlc
```

CI'da: `sqlc generate && git diff --exit-code` — generatsiya eskirmaganini tekshiradi. `sqlc/` papkasi commit qilinadi.

## Cheklovlar va yechimlar

| Muammo | Yechim |
|--------|--------|
| Dinamik `WHERE` (ixtiyoriy filterlar) | `sqlc.narg` + `IS NULL OR` pattern (yuqorida). Juda murakkab bo'lsa — o'sha bitta query'ni `squirrel`/qo'lda yozing, qolgani sqlc |
| Dinamik `ORDER BY` | `CASE WHEN @sort = 'name' THEN name END` yoki qo'lda |
| `IN (...)` ro'yxat | `WHERE id = ANY($1::bigint[])` — `[]int64` qabul qiladi |
| `NUMERIC` → `pgtype.Numeric` qulay emas | `overrides` bilan `float64`/`decimal.Decimal` |
| NULL ustunlar `pgtype.Text` | `emit_pointers_for_null_types: true` → `*string` |
| Ustun qo'shildi, kod eskirdi | `sqlc generate` — kompilyator o'zgarishi kerak joylarni ko'rsatadi |
| goose `-- +goose` kommentlari | sqlc tushunadi, `Down` qismini e'tiborsiz qoldiradi |

## Qachon sqlc, qachon qo'lda pgx

- **sqlc**: ko'p CRUD, statik query'lar, tip xavfsizligi muhim, jamoa SQL'ni yaxshi biladi. Ko'pchilik yangi Go+Postgres loyihalarda default tanlov.
- **Qo'lda pgx** (`CollectRows`): juda dinamik query'lar, o'rganish bosqichi (avval `Scan` nima ekanini tushunib oling).
- Ikkalasini **aralash** ishlatish normal: 90% sqlc, 10% qo'lda.

O'rganish tartibi: avval 06-repository.md bo'yicha qo'lda 1 ta repository yozing → keyin xuddi shuni sqlc bilan qayta yozing. Farqni o'zingiz his qilasiz.
