# 02 — Papka tuzilmasi, nomlash va import qoidalari

## Tuzilma (layer-based Clean Architecture)

```
myservice/
├── cmd/
│   └── api/
│       └── main.go                  # faqat: config o'qi → app qur (wire) → ishga tushir
├── internal/                        # tashqi moduldan import qilib bo'lmaydi (Go qoidasi)
│   ├── app/
│   │   ├── app.go                   # App struct: server, pool, logger; Run/Shutdown
│   │   ├── wire.go                  # wire inject funksiyasi (//go:build wireinject)
│   │   └── wire_gen.go              # wire generatsiya qiladi (qo'lda yozilmaydi)
│   ├── config/
│   │   └── config.go
│   ├── domain/                      # ENG ICHKI qatlam: entity + domain error + qoidalar
│   │   ├── user.go
│   │   ├── product.go
│   │   └── errors.go
│   ├── service/                     # use case (biznes mantiq)
│   │   ├── ports.go                 # service'ga kerak interfeyslar (repository, hasher, token...)
│   │   ├── auth_service.go
│   │   ├── auth_service_test.go
│   │   ├── product_service.go
│   │   └── product_service_test.go
│   ├── repository/
│   │   └── postgres/
│   │       ├── user_repository.go
│   │       ├── product_repository.go
│   │       ├── product_repository_test.go   # integration (testcontainers)
│   │       └── tx.go
│   └── transport/
│       └── http/
│           ├── router.go            # barcha route'lar shu yerda
│           ├── server.go            # http.Server sozlamalari
│           ├── dto/
│           │   ├── auth_dto.go
│           │   └── product_dto.go   # request/response + mapper
│           ├── handler/
│           │   ├── auth_handler.go
│           │   ├── product_handler.go
│           │   └── product_handler_test.go
│           ├── middleware/
│           │   ├── auth.go          # JWT tekshirish, rol tekshirish
│           │   ├── logger.go
│           │   └── request_id.go
│           └── response/
│               ├── response.go      # JSON(), Error()
│               └── errors.go        # domain error → HTTP status
├── pkg/                             # loyihaga bog'liq BO'LMAGAN kod (boshqa loyihada ham ishlaydi)
│   ├── logger/
│   ├── postgres/
│   ├── validator/
│   ├── jwt/
│   └── hash/
├── migrations/
│   ├── 000001_create_users.up.sql
│   └── 000001_create_users.down.sql
├── api/
│   └── swagger/                     # swag generatsiya qiladi
├── docs/                            # qo'llanmalar
├── mocks/                           # mockery generatsiya qiladi
├── .env.example
├── .mockery.yaml
├── .golangci.yml
├── Makefile
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

## Qatlamlar va bog'liqlik yo'nalishi

```
 transport/http ──▶ service ──▶ domain ◀── repository/postgres
      (handler)      (use case)   (entity)      (SQL)
```

| Qatlam | Nimani biladi | Nimani BILMAYDI |
|--------|---------------|-----------------|
| `domain` | faqat o'zini (std lib) | HTTP, SQL, JSON teglar |
| `service` | domain + o'zi e'lon qilgan interfeyslar | pgx, chi, http |
| `repository` | domain + pgx | service, http |
| `transport/http` | service + dto + domain error | pgx, SQL |
| `app` | hammasini (faqat yig'ish uchun) | — |

**Qoida:** ichki qatlam tashqi qatlamni import qilmaydi. `domain` paketida `import "net/http"` ko'rsangiz — xato.

## `internal/` vs `pkg/`

- `internal/` — faqat shu moduldan import qilinadi (kompilyator tekshiradi). Biznes kod hammasi shu yerda.
- `pkg/` — boshqa loyihaga ko'chirsangiz ham ishlaydigan kod (`logger`, `postgres` pool, `validator`, `jwt`). Agar ichida `domain` import qilinsa — u `pkg` emas, `internal`ga o'tkazing.

## Nomlash qoidalari

| Narsa | Qoida | Misol |
|-------|-------|-------|
| Papka / paket | kichik harf, bitta so'z, `_` yo'q | `handler`, `postgres`, `dto` |
| Fayl | `snake_case.go` | `product_service.go`, `request_id.go` |
| Test fayl | `<fayl>_test.go` | `product_service_test.go` |
| Struct | `PascalCase` | `handler.ProductHandler`, `service.ProductService` |
| Interfeys | rol nomi yoki fe'l+`er` | `ProductRepository`, `PasswordHasher`, `TokenManager` |
| Konstruktor | `New<Type>` | `NewProductService(repo ProductRepository)` |
| Xato o'zgaruvchi | `Err` prefiks | `ErrProductNotFound` |
| Private struct | `camelCase` | `productRow` |

**Stuttering'dan qoching:** `postgres.PostgresProductRepository` → `postgres.ProductRepository`.

## Import qoidalari

Uch guruh, bo'sh qator bilan ajratilgan (`goimports` avtomatik tartiblaydi):

```go
import (
    // 1. standart kutubxona
    "context"
    "errors"
    "fmt"

    // 2. tashqi kutubxonalar
    "github.com/go-chi/chi/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    // 3. shu loyiha
    "github.com/yodzafar/myservice/internal/domain"
    "github.com/yodzafar/myservice/internal/service"
    httptransport "github.com/yodzafar/myservice/internal/transport/http" // alias: net/http bilan to'qnashadi
)
```

- Alias faqat nom to'qnashganda yoki noaniq bo'lganda.
- Dot-import (`import . "..."`) ishlatilmaydi.
- Blank import (`_ "..."`) faqat side-effect uchun: swagger docs, sql driver.

## Fayllar bir-biriga qanday ulanadi (Product misolida)

```
domain/product.go       type Product struct {...}
domain/errors.go        var ErrProductNotFound = errors.New("product not found")
        ▲
service/ports.go        type ProductRepository interface { Create(ctx, *domain.Product) error; ... }
service/product_service.go
                        type ProductService struct { repo ProductRepository }  → NewProductService(repo)
        ▲                                   ▲
repository/postgres/    type ProductRepository struct { db *pgxpool.Pool } → NewProductRepository(db)
product_repository.go   (service.ProductRepository ni implement qiladi — compile-time tekshiruv bilan)

transport/http/handler/ type ProductHandler struct { svc *service.ProductService } → NewProductHandler(svc)
transport/http/router.go NewRouter(deps) http.Handler
app/wire.go             hammasini bog'laydi
cmd/api/main.go         app.InitApp(cfg) → app.Run()
```

## Loyiha kattalashsa — feature-based

10+ entity bo'lsa qatlam bo'yicha emas, feature bo'yicha bo'ling:

```
internal/product/{domain.go, service.go, repository.go, handler.go, dto.go}
internal/user/{...}
internal/order/{...}
```

O'rganish uchun **layer-based** tuzilma tavsiya — qatlamlarni aniq ko'rasiz.
