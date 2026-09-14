# 06 — Repository qatlami

Repository — entity'ni saqlash/olish. Faqat SQL biladi, biznes mantiq yo'q.

## 1. Interfeys — iste'molchi (service) e'lon qiladi

`internal/service/ports.go`:

```go
package service

import (
    "context"

    "github.com/yodzafar/myservice/internal/domain"
)

type UserRepository interface {
    Create(ctx context.Context, u *domain.User) error
    GetByID(ctx context.Context, id int64) (*domain.User, error)
    GetByEmail(ctx context.Context, email string) (*domain.User, error)
    List(ctx context.Context, limit, offset int) ([]domain.User, int, error)
    UpdatePassword(ctx context.Context, id int64, passwordHash string, now time.Time) error
    UpdateRole(ctx context.Context, id int64, role domain.Role, now time.Time) error
    Delete(ctx context.Context, id int64) error
}

type ProductRepository interface {
    Create(ctx context.Context, p *domain.Product) error
    GetByID(ctx context.Context, id int64) (*domain.Product, error)
    Update(ctx context.Context, p *domain.Product) error
    List(ctx context.Context, f ProductFilter) ([]domain.Product, int, error)
    Delete(ctx context.Context, id int64) error
}

type ProductFilter struct {
    OwnerID *int64
    Status  *domain.ProductStatus
    Limit   int
    Offset  int
}
```

Nega service'da? Service "menga shu metodlar kerak" deydi; repository "men bera olaman" deb implement qiladi. Aks holda service repository paketiga bog'lanib qoladi.

## 2. Implementatsiya — `internal/repository/postgres/product_repository.go`

```go
package postgres

import (
    "context"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/jackc/pgx/v5"

    "github.com/yodzafar/myservice/internal/domain"
    "github.com/yodzafar/myservice/internal/service"
)

// Compile-time tekshiruv
var _ service.ProductRepository = (*ProductRepository)(nil)

type ProductRepository struct {
    db DB // tx.go dagi interfeys — pool ham, tx ham bo'la oladi
}

func NewProductRepository(db DB) *ProductRepository {
    return &ProductRepository{db: db}
}

// productRow — DB qatori. Domain va jadval 1:1 bo'lsa bu struct shart emas:
// domain.Product'ga `db` teg qo'yib, to'g'ridan-to'g'ri RowToStructByName[domain.Product] ishlating
// (05-domain.md). Bu misol "ajratilgan" variantni ko'rsatadi.
type productRow struct {
    ID        int64     `db:"id"`
    OwnerID   int64     `db:"owner_id"`
    Name      string    `db:"name"`
    Price     float64   `db:"price"`
    Status    string    `db:"status"`
    CreatedAt time.Time `db:"created_at"`
    UpdatedAt time.Time `db:"updated_at"`
}

func (r productRow) toDomain() domain.Product {
    return domain.Product{
        ID: r.ID, OwnerID: r.OwnerID, Name: r.Name, Price: r.Price,
        Status: domain.ProductStatus(r.Status), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
    }
}

const productColumns = `id, owner_id, name, price, status, created_at, updated_at`

func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
    const q = `
        INSERT INTO products (owner_id, name, price, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id`
    err := r.db.QueryRow(ctx, q, p.OwnerID, p.Name, p.Price, p.Status, p.CreatedAt, p.UpdatedAt).Scan(&p.ID)
    if err != nil {
        return fmt.Errorf("product repo: create: %w", err)
    }
    return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
    rows, err := r.db.Query(ctx, `SELECT `+productColumns+` FROM products WHERE id = $1`, id)
    if err != nil {
        return nil, fmt.Errorf("product repo: get: %w", err)
    }
    row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[productRow])
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, domain.ErrProductNotFound
        }
        return nil, fmt.Errorf("product repo: get: %w", err)
    }
    p := row.toDomain()
    return &p, nil
}

func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
    const q = `UPDATE products SET name=$1, price=$2, status=$3, updated_at=$4 WHERE id=$5`
    tag, err := r.db.Exec(ctx, q, p.Name, p.Price, p.Status, p.UpdatedAt, p.ID)
    if err != nil {
        return fmt.Errorf("product repo: update: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return domain.ErrProductNotFound
    }
    return nil
}

// List — dinamik filter. Placeholder raqamlarini qo'lda yig'amiz.
func (r *ProductRepository) List(ctx context.Context, f service.ProductFilter) ([]domain.Product, int, error) {
    where := []string{"1=1"}
    args := []any{}
    if f.OwnerID != nil {
        args = append(args, *f.OwnerID)
        where = append(where, fmt.Sprintf("owner_id = $%d", len(args)))
    }
    if f.Status != nil {
        args = append(args, *f.Status)
        where = append(where, fmt.Sprintf("status = $%d", len(args)))
    }
    cond := strings.Join(where, " AND ")

    var total int
    if err := r.db.QueryRow(ctx, `SELECT count(*) FROM products WHERE `+cond, args...).Scan(&total); err != nil {
        return nil, 0, fmt.Errorf("product repo: count: %w", err)
    }

    args = append(args, f.Limit, f.Offset)
    q := fmt.Sprintf(`SELECT %s FROM products WHERE %s ORDER BY id DESC LIMIT $%d OFFSET $%d`,
        productColumns, cond, len(args)-1, len(args))
    rows, err := r.db.Query(ctx, q, args...)
    if err != nil {
        return nil, 0, fmt.Errorf("product repo: list: %w", err)
    }
    rowsData, err := pgx.CollectRows(rows, pgx.RowToStructByName[productRow])
    if err != nil {
        return nil, 0, fmt.Errorf("product repo: list: %w", err)
    }
    out := make([]domain.Product, 0, len(rowsData))
    for _, rw := range rowsData {
        out = append(out, rw.toDomain())
    }
    return out, total, nil
}

func (r *ProductRepository) Delete(ctx context.Context, id int64) error {
    tag, err := r.db.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
    if err != nil {
        return fmt.Errorf("product repo: delete: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return domain.ErrProductNotFound
    }
    return nil
}
```

Dinamik `WHERE` ko'payib ketsa — `github.com/Masterminds/squirrel` query builder ishlating.

SQL, `Scan` va `productRow`ni qo'lda yozmaslik uchun — [23-sqlc.md](23-sqlc.md) (avval qo'lda bir marta yozib ko'ring).

## Mapper'larni alohida faylga — `postgres/mapper.go`

Repository faylida faqat SQL/mantiq qolsin. Mapping — o'sha **paket ichida** alohida fayl (umumiy `mapper` paketi emas: u ikki qatlamni ham import qiladi, bog'liqlik yo'nalishi buziladi).

```go
package postgres

// ---- Product ----

func toDomainProduct(r productRow) domain.Product { // sqlc bo'lsa: r sqlc.Product
    return domain.Product{
        ID: r.ID, OwnerID: r.OwnerID, Name: r.Name, Price: r.Price,
        Status: domain.ProductStatus(r.Status), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
    }
}

// sqlc bilan: domain → insert params
func toCreateProductParams(p *domain.Product) sqlc.CreateProductParams {
    return sqlc.CreateProductParams{
        OwnerID: p.OwnerID, Name: p.Name, Price: p.Price, Status: string(p.Status),
        CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
    }
}

// ---- User ----

func toDomainUser(r userRow) domain.User { ... }
```

Entity ko'p bo'lsa: `product_mapper.go`, `user_mapper.go`. Nomlash: `toDomainX` (DB → domain), `toXParams` (domain → DB).

Slice mapping uchun har entity'ga `toDomainProducts` yozmang — bitta generic yordamchi:

```go
// pkg/slices/slices.go
package slices

func Map[T, R any](in []T, fn func(T) R) []R {
    out := make([]R, 0, len(in)) // bo'sh bo'lsa nil emas, []
    for _, v := range in {
        out = append(out, fn(v))
    }
    return out
}

// repository'da
return slices.Map(rows, toDomainProduct), total, nil
```

## `user_repository.go` — to'liq misol (domain'ga `db` teg, row struct'siz)

`domain.User`da `db` teglar bor ([05-domain.md](05-domain.md), 1-bosqich) — alohida `userRow` kerak emas, `RowToStructByName[domain.User]` to'g'ridan-to'g'ri ishlaydi.

Interfeys (`service/user_ports.go`):

```go
type UserRepository interface {
    Create(ctx context.Context, u *domain.User) error
    GetByID(ctx context.Context, id int64) (*domain.User, error)
    GetByEmail(ctx context.Context, email string) (*domain.User, error)
    List(ctx context.Context, limit, offset int) ([]domain.User, int, error)
    UpdatePassword(ctx context.Context, id int64, passwordHash string, now time.Time) error
    UpdateRole(ctx context.Context, id int64, role domain.Role, now time.Time) error
    Delete(ctx context.Context, id int64) error
}
```

```go
package postgres

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"

    "github.com/yodzafar/myservice/internal/domain"
    "github.com/yodzafar/myservice/internal/service"
)

var _ service.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
    db DB // tx.go dagi interfeys: pool yoki tx
}

func NewUserRepository(db DB) *UserRepository {
    return &UserRepository{db: db}
}

const userColumns = `id, email, password_hash, role, created_at, updated_at`

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
    const q = `
        INSERT INTO users (email, password_hash, role, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id`

    err := r.db.QueryRow(ctx, q, u.Email, u.PasswordHash, u.Role, u.CreatedAt, u.UpdatedAt).Scan(&u.ID)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation (email)
            return domain.ErrEmailTaken
        }
        return fmt.Errorf("user repo: create: %w", err)
    }
    return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
    return r.getOne(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
    return r.getOne(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email)
}

// getOne — bitta qator o'qish; ikkala Get uchun umumiy
func (r *UserRepository) getOne(ctx context.Context, q string, args ...any) (*domain.User, error) {
    rows, err := r.db.Query(ctx, q, args...)
    if err != nil {
        return nil, fmt.Errorf("user repo: query: %w", err)
    }
    u, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.User])
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, domain.ErrUserNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("user repo: scan: %w", err)
    }
    return &u, nil
}

func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]domain.User, int, error) {
    var total int
    if err := r.db.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&total); err != nil {
        return nil, 0, fmt.Errorf("user repo: count: %w", err)
    }

    rows, err := r.db.Query(ctx,
        `SELECT `+userColumns+` FROM users ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset)
    if err != nil {
        return nil, 0, fmt.Errorf("user repo: list: %w", err)
    }
    users, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.User])
    if err != nil {
        return nil, 0, fmt.Errorf("user repo: list scan: %w", err)
    }
    return users, total, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string, now time.Time) error {
    return r.exec(ctx, `UPDATE users SET password_hash = $2, updated_at = $3 WHERE id = $1`, id, passwordHash, now)
}

func (r *UserRepository) UpdateRole(ctx context.Context, id int64, role domain.Role, now time.Time) error {
    return r.exec(ctx, `UPDATE users SET role = $2, updated_at = $3 WHERE id = $1`, id, role, now)
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
    return r.exec(ctx, `DELETE FROM users WHERE id = $1`, id)
}

// exec — UPDATE/DELETE; 0 qator = not found
func (r *UserRepository) exec(ctx context.Context, q string, args ...any) error {
    tag, err := r.db.Exec(ctx, q, args...)
    if err != nil {
        return fmt.Errorf("user repo: exec: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return domain.ErrUserNotFound
    }
    return nil
}
```

E'tibor bering:
- `domain.Role` (`type Role string`) — pgx uni `text` sifatida yozadi/o'qiydi, konvertatsiya kerak emas.
- `getOne` / `exec` yordamchilari takrorlanishni yig'adi — `GetByID`, `GetByEmail`, uchta yozuv metodi bir qatorga tushdi.
- `PasswordHash` domain'da bor, `dto.UserResponse`ga hech qachon o'tmaydi.
- `UpdatePassword`/`UpdateRole` butun `*domain.User` emas, faqat kerakli field'larni oladi — tasodifan boshqa ustunlarni qayta yozmaysiz.

## Qoidalar

1. **Infra xatosini domain xatosiga tarjima qiling** (`pgx.ErrNoRows` → `domain.ErrProductNotFound`, `23505` → `ErrEmailTaken`).
2. Boshqa xatolarni **wrap** qiling: `fmt.Errorf("product repo: create: %w", err)`.
3. `SELECT *` yozmang — ustun tartibi o'zgarsa buziladi. Ustunlarni konstantaga chiqaring.
4. `RowsAffected() == 0` → not found (Update/Delete).
5. Repository **biznes qaror qabul qilmaydi**: "foydalanuvchi bu mahsulotni o'zgartira oladimi" — service.
6. Bir entity = bir fayl. 300+ qator bo'lsa `product_repository_read.go` / `_write.go`.
7. Row struct (`productRow`) — faqat domain va jadval shakli farqlanganda (JOIN, value object, `jsonb`). 1:1 bo'lsa `domain.Product`ga `db` teg yetadi — qaysi entity uchun kerak bo'lsa o'shanga yozing, hammasiga emas.

Test — integration (real Postgres): [15-testing.md](15-testing.md).
