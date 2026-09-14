# 07 — Service (use case) qatlami

Service — biznes mantiq. HTTP'ni ham, SQL'ni ham bilmaydi. Faqat domain + `ports.go`dagi interfeyslar.

## `internal/service/ports.go` — barcha tashqi bog'liqliklar

```go
package service

import (
    "context"
    "time"

    "github.com/yodzafar/myservice/internal/domain"
)

// Repository interfeyslari — 06-repository.md da
type UserRepository interface { /* ... */ }
type ProductRepository interface { /* ... */ }

// Kichik interfeyslar — testda mock qilish oson
type PasswordHasher interface {
    Hash(password string) (string, error)
    Compare(hash, password string) error
}

type TokenManager interface {
    GenerateAccess(userID int64, role domain.Role) (string, error)
    GenerateRefresh() (string, error)
    ParseAccess(token string) (*TokenClaims, error)
}

type TokenClaims struct {
    UserID int64
    Role   domain.Role
}

type Clock interface {
    Now() time.Time
}
```

## Interfeyslarni qayerga yozish — `ports.go` shartmi?

Yo'q. Qoida bitta: **interfeys iste'molchi paketida (`service`) e'lon qilinadi**. Qaysi faylda turishi hajmga bog'liq.

| Variant | Qachon | Tuzilma |
|---------|--------|---------|
| Bitta `ports.go` | 3–5 interfeys, 1–2 service (boshlang'ich) | `service/ports.go` |
| Entity bo'yicha fayl | o'rta loyiha — **tavsiya** | `product_ports.go`, `auth_ports.go`, umumiylar (`Clock`, `TxManager`) `ports.go`da |
| Service fayli ichida | Go idiomasi, fayl uzun bo'lmasa | `product_service.go` boshida `type ProductRepository interface` |

```
internal/service/
├── ports.go               # umumiy: Clock, TxManager
├── product_service.go
├── product_ports.go       # ProductRepository, ProductCache
├── auth_service.go
├── auth_ports.go          # UserRepository, PasswordHasher, TokenManager, RefreshTokenStore
├── order_service.go
└── order_ports.go         # OrderRepository, PaymentGateway, EventPublisher
```

### Ko'p interfeys — normal. Katta interfeys — muammo

Go'da interfeys qancha kichik bo'lsa shuncha yaxshi (`io.Reader` — 1 metod). Har service **faqat o'ziga kerak** metodlarni e'lon qiladi:

```go
// Yomon: bitta ulkan interfeys, hamma hammasini "biladi"
type Repository interface {
    CreateUser(...); GetUser(...); CreateProduct(...); ListProducts(...); CreateOrder(...) // 30 metod
}

// Yaxshi: kichik, rol bo'yicha
type ProductReader interface {
    GetByID(ctx context.Context, id int64) (*domain.Product, error)
}
type ProductWriter interface {
    Create(ctx context.Context, p *domain.Product) error
    Update(ctx context.Context, p *domain.Product) error
}
type ProductRepository interface { // kerak bo'lsa birlashtirish (embedding)
    ProductReader
    ProductWriter
}
```

- `OrderService`ga faqat mahsulotni o'qish kerak bo'lsa — `ProductReader` oladi, 30 metodli mock yozmaydi.
- Bitta `*postgres.ProductRepository` struct hammasini implement qiladi; interfeyslar faqat "kim nimani ko'radi"ni cheklaydi.
- Wire'da: `wire.Bind(new(service.ProductReader), new(*postgres.ProductRepository))` — bitta struct bir nechta interfeysga bind qilinadi.

10+ entity bo'lsa `service/` paketi 40+ faylga o'sadi — o'shanda feature-based tuzilmaga o'ting ([02-folder-structure.md](02-folder-structure.md) oxiri): `internal/product/ports.go` faqat product interfeyslarini saqlaydi.

## `internal/service/product_service.go`

```go
package service

import (
    "context"
    "fmt"
    "log/slog"

    "github.com/yodzafar/myservice/internal/domain"
)

type ProductService struct {
    repo  ProductRepository
    clock Clock
    log   *slog.Logger
}

func NewProductService(repo ProductRepository, clock Clock, log *slog.Logger) *ProductService {
    return &ProductService{repo: repo, clock: clock, log: log}
}

// Input struct'lar — DTO emas. Transport DTO'ni shunga map qiladi.
type CreateProductInput struct {
    OwnerID int64
    Name    string
    Price   float64
}

type UpdateProductInput struct {
    Name  *string  // nil = o'zgartirilmaydi (partial update)
    Price *float64
}

func (s *ProductService) Create(ctx context.Context, in CreateProductInput) (*domain.Product, error) {
    p, err := domain.NewProduct(in.OwnerID, in.Name, in.Price, s.clock.Now())
    if err != nil {
        return nil, err
    }
    if err := s.repo.Create(ctx, p); err != nil {
        return nil, fmt.Errorf("create product: %w", err)
    }
    return p, nil
}

func (s *ProductService) Get(ctx context.Context, id int64) (*domain.Product, error) {
    return s.repo.GetByID(ctx, id)
}

// actor — kim so'rayapti (auth middleware'dan keladi). Ruxsat shu yerda tekshiriladi.
func (s *ProductService) Update(ctx context.Context, actor *domain.User, id int64, in UpdateProductInput) (*domain.Product, error) {
    p, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    if !p.CanBeEditedBy(actor) {
        return nil, domain.ErrForbidden
    }
    if in.Name != nil {
        p.Name = *in.Name
    }
    if in.Price != nil {
        if *in.Price < 0 {
            return nil, domain.ErrInvalidPrice
        }
        p.Price = *in.Price
    }
    p.UpdatedAt = s.clock.Now()

    if err := s.repo.Update(ctx, p); err != nil {
        return nil, fmt.Errorf("update product: %w", err)
    }
    return p, nil
}

func (s *ProductService) Publish(ctx context.Context, actor *domain.User, id int64) (*domain.Product, error) {
    p, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    if !p.CanBeEditedBy(actor) {
        return nil, domain.ErrForbidden
    }
    if err := p.Publish(s.clock.Now()); err != nil { // biznes qoida — domain'da
        return nil, err
    }
    if err := s.repo.Update(ctx, p); err != nil {
        return nil, fmt.Errorf("publish product: %w", err)
    }
    s.log.InfoContext(ctx, "product published", "product_id", p.ID, "actor_id", actor.ID)
    return p, nil
}

type ListProductsInput struct {
    OwnerID *int64
    Status  *domain.ProductStatus
    Page    int
    PerPage int
}

func (s *ProductService) List(ctx context.Context, in ListProductsInput) ([]domain.Product, int, error) {
    if in.Page < 1 {
        in.Page = 1
    }
    if in.PerPage < 1 || in.PerPage > 100 {
        in.PerPage = 20
    }
    return s.repo.List(ctx, ProductFilter{
        OwnerID: in.OwnerID, Status: in.Status,
        Limit: in.PerPage, Offset: (in.Page - 1) * in.PerPage,
    })
}

func (s *ProductService) Delete(ctx context.Context, actor *domain.User, id int64) error {
    p, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return err
    }
    if !p.CanBeEditedBy(actor) {
        return domain.ErrForbidden
    }
    return s.repo.Delete(ctx, id)
}
```

Auth service (`Register`, `Login`, `Refresh`) — [12-auth-jwt-roles.md](12-auth-jwt-roles.md).

## `pkg/clock/clock.go`

```go
package clock

import "time"

type Real struct{}

func (Real) Now() time.Time { return time.Now() }
```

Testda: `type fakeClock struct{ t time.Time }; func (f fakeClock) Now() time.Time { return f.t }`.

## Qoidalar

1. **Kirish = Input struct**, chiqish = domain entity. HTTP DTO service'ga kirmaydi.
2. Service xato turini o'zgartirmaydi: domain xatosini qaytaradi yoki wrap qiladi. Transport HTTP statusga tarjima qiladi.
3. Tashqi dunyo (vaqt, random, DB, hash) — interfeys orqali → unit test deterministik.
4. **Ruxsat (authorization) service'da**: middleware "kim" ekanini aniqlaydi (authentication), service "mumkinmi" deb tekshiradi.
5. Partial update — pointer field'lar (`*string`): nil = tegilmadi.
6. Log faqat qaror qabul qilingan joyda. Har bir request'ni middleware log qiladi.
7. Bitta service = bitta aggregate. `ProductService` ichida `UserRepository` kerak bo'lsa — interfeys orqali, lekin bu signal: balki alohida use case kerak.
