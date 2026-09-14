# 05 — Domain qatlami

Domain — loyihaning yuragi: entity, domain xatolar, sof biznes qoidalar. HTTP, SQL, JSON teg **yo'q**.

## `internal/domain/user.go`

```go
package domain

import "time"

type Role string

const (
    RoleUser  Role = "user"
    RoleAdmin Role = "admin"
)

func (r Role) IsValid() bool {
    return r == RoleUser || r == RoleAdmin
}

type User struct {
    ID           int64
    Email        string
    PasswordHash string
    Role         Role
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }
```

## `internal/domain/product.go`

```go
package domain

import (
    "strings"
    "time"
)

type ProductStatus string

const (
    ProductDraft     ProductStatus = "draft"
    ProductPublished ProductStatus = "published"
)

type Product struct {
    ID        int64
    OwnerID   int64
    Name      string
    Price     float64 // pul uchun real loyihada int64 (tiyin) yoki decimal ishlating
    Status    ProductStatus
    CreatedAt time.Time
    UpdatedAt time.Time
}

// NewProduct — entity'ni to'g'ri holatda yaratadi (invariantlar shu yerda).
func NewProduct(ownerID int64, name string, price float64, now time.Time) (*Product, error) {
    name = strings.TrimSpace(name)
    if name == "" {
        return nil, ErrInvalidProductName
    }
    if price < 0 {
        return nil, ErrInvalidPrice
    }
    return &Product{
        OwnerID: ownerID, Name: name, Price: price, Status: ProductDraft,
        CreatedAt: now, UpdatedAt: now,
    }, nil
}

// Biznes qoidalar — entity metodlari
func (p *Product) Publish(now time.Time) error {
    if p.Status == ProductPublished {
        return ErrAlreadyPublished
    }
    p.Status = ProductPublished
    p.UpdatedAt = now
    return nil
}

func (p *Product) CanBeEditedBy(u *User) bool {
    return u.IsAdmin() || p.OwnerID == u.ID
}
```

## `internal/domain/errors.go`

```go
package domain

import "errors"

var (
    // umumiy
    ErrNotFound  = errors.New("not found")
    ErrForbidden = errors.New("forbidden")

    // user / auth
    ErrUserNotFound       = errors.New("user not found")
    ErrEmailTaken         = errors.New("email already taken")
    ErrInvalidCredentials = errors.New("invalid email or password")
    ErrInvalidToken       = errors.New("invalid or expired token")

    // product
    ErrProductNotFound    = errors.New("product not found")
    ErrInvalidProductName = errors.New("product name is empty")
    ErrInvalidPrice       = errors.New("price must be >= 0")
    ErrAlreadyPublished   = errors.New("product already published")
)
```

Bu xatolar butun loyihada **sentinel**:
- repository: `pgx.ErrNoRows` → `domain.ErrProductNotFound`
- service: `errors.Is(err, domain.ErrProductNotFound)`
- transport: `domain.ErrProductNotFound` → `404`

## Qoidalar

1. **JSON/DB teglari domain'da yo'q.** `json:` — DTO'da, `db:` — repository'dagi private row struct'da.
2. **Vaqtni tashqaridan bering** (`now time.Time`) — testda deterministik.
3. **Entity o'zini himoya qiladi**: noto'g'ri `Product` yaratib bo'lmasin.
4. Enum → `type X string` + konstantalar + `IsValid()`.
5. Pointer faqat "yo'q bo'lishi mumkin" degani (`*time.Time` — `DeletedAt`).
6. Domain testi eng oson — bog'liqlik yo'q:

```go
func TestProduct_Publish_Twice(t *testing.T) {
    now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
    p, _ := domain.NewProduct(1, "Phone", 100, now)
    require.NoError(t, p.Publish(now))
    require.ErrorIs(t, p.Publish(now), domain.ErrAlreadyPublished)
}
```
