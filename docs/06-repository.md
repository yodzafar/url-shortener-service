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

// productRow — DB qatori. domain.Product bilan aralashtirmang.
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

## `user_repository.go` (qisqa — unique xato tarjimasi)

```go
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
    const q = `INSERT INTO users (email, password_hash, role, created_at, updated_at)
               VALUES ($1, $2, $3, $4, $5) RETURNING id`
    err := r.db.QueryRow(ctx, q, u.Email, u.PasswordHash, u.Role, u.CreatedAt, u.UpdatedAt).Scan(&u.ID)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            return domain.ErrEmailTaken
        }
        return fmt.Errorf("user repo: create: %w", err)
    }
    return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
    rows, err := r.db.Query(ctx, `SELECT id, email, password_hash, role, created_at, updated_at
                                  FROM users WHERE email = $1`, email)
    if err != nil {
        return nil, fmt.Errorf("user repo: get by email: %w", err)
    }
    row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[userRow])
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, domain.ErrUserNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("user repo: get by email: %w", err)
    }
    u := row.toDomain()
    return &u, nil
}
```

## Qoidalar

1. **Infra xatosini domain xatosiga tarjima qiling** (`pgx.ErrNoRows` → `domain.ErrProductNotFound`, `23505` → `ErrEmailTaken`).
2. Boshqa xatolarni **wrap** qiling: `fmt.Errorf("product repo: create: %w", err)`.
3. `SELECT *` yozmang — ustun tartibi o'zgarsa buziladi. Ustunlarni konstantaga chiqaring.
4. `RowsAffected() == 0` → not found (Update/Delete).
5. Repository **biznes qaror qabul qilmaydi**: "foydalanuvchi bu mahsulotni o'zgartira oladimi" — service.
6. Bir entity = bir fayl. 300+ qator bo'lsa `product_repository_read.go` / `_write.go`.

Test — integration (real Postgres): [15-testing.md](15-testing.md).
