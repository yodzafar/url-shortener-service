# 15 — Testing

## Uch daraja

| Daraja | Nima | Bog'liqlik | Tezlik | Qayerda |
|--------|------|-----------|--------|---------|
| **Unit** | service, domain | mock | ms | `service/*_test.go` |
| **Handler** | HTTP → JSON | mock service | ms | `handler/*_test.go` |
| **Integration** | repository | real Postgres (testcontainers) | sek | `repository/postgres/*_test.go` |
| **E2E** (ixtiyoriy) | butun app | real DB + HTTP | sek | `tests/e2e/` |

Kutubxonalar: `github.com/stretchr/testify` (assert/require), `github.com/vektra/mockery/v2` (mock generatsiya), `github.com/testcontainers/testcontainers-go`.

## Mock generatsiya — mockery

`.mockery.yaml` (loyiha ildizi):

```yaml
with-expecter: true
dir: mocks
outpkg: mocks
mockname: "{{.InterfaceName}}"
filename: "{{.InterfaceName | snakecase}}.go"
packages:
  github.com/yodzafar/myservice/internal/service:
    interfaces:
      UserRepository:
      ProductRepository:
      PasswordHasher:
      TokenManager:
      RefreshTokenStore:
      Clock:
```

```bash
mockery          # mocks/product_repository.go va h.k. yaratiladi
```

Alternativa: `go.uber.org/mock` (`mockgen`). Ikkalasi ham yaxshi; mockery `EXPECT()` sintaksisi o'qish uchun qulay.

## 1. Unit test — service

`internal/service/product_service_test.go`:

```go
package service_test

import (
    "context"
    "log/slog"
    "testing"
    "time"

    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "github.com/yodzafar/myservice/internal/domain"
    "github.com/yodzafar/myservice/internal/service"
    "github.com/yodzafar/myservice/mocks"
)

var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// Har test uchun toza bog'liqliklar
func newProductService(t *testing.T) (*service.ProductService, *mocks.ProductRepository) {
    t.Helper()
    repo := mocks.NewProductRepository(t) // t berilsa: test oxirida kutilgan chaqiriqlar tekshiriladi
    clk := mocks.NewClock(t)
    clk.EXPECT().Now().Return(fixedNow).Maybe()
    svc := service.NewProductService(repo, clk, slog.Default())
    return svc, repo
}

func TestProductService_Create(t *testing.T) {
    svc, repo := newProductService(t)

    repo.EXPECT().
        Create(mock.Anything, mock.MatchedBy(func(p *domain.Product) bool {
            return p.Name == "Phone" && p.Status == domain.ProductDraft
        })).
        RunAndReturn(func(_ context.Context, p *domain.Product) error {
            p.ID = 42 // DB ID berganini simulyatsiya
            return nil
        })

    p, err := svc.Create(context.Background(), service.CreateProductInput{OwnerID: 1, Name: "Phone", Price: 10})

    require.NoError(t, err)
    require.Equal(t, int64(42), p.ID)
    require.Equal(t, fixedNow, p.CreatedAt)
}

// Table-driven — bir nechta holat bitta testda
func TestProductService_Update(t *testing.T) {
    owner := &domain.User{ID: 1, Role: domain.RoleUser}
    other := &domain.User{ID: 2, Role: domain.RoleUser}
    admin := &domain.User{ID: 3, Role: domain.RoleAdmin}
    existing := func() *domain.Product { return &domain.Product{ID: 10, OwnerID: 1, Name: "Old", Price: 5} }
    newName := "New"

    tests := []struct {
        name    string
        actor   *domain.User
        setup   func(repo *mocks.ProductRepository)
        wantErr error
    }{
        {
            name:  "owner can update",
            actor: owner,
            setup: func(r *mocks.ProductRepository) {
                r.EXPECT().GetByID(mock.Anything, int64(10)).Return(existing(), nil)
                r.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)
            },
        },
        {
            name:  "admin can update",
            actor: admin,
            setup: func(r *mocks.ProductRepository) {
                r.EXPECT().GetByID(mock.Anything, int64(10)).Return(existing(), nil)
                r.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)
            },
        },
        {
            name:  "other user forbidden",
            actor: other,
            setup: func(r *mocks.ProductRepository) {
                r.EXPECT().GetByID(mock.Anything, int64(10)).Return(existing(), nil)
                // Update chaqirilmasligi kerak — mockery buni tekshiradi
            },
            wantErr: domain.ErrForbidden,
        },
        {
            name:  "not found",
            actor: owner,
            setup: func(r *mocks.ProductRepository) {
                r.EXPECT().GetByID(mock.Anything, int64(10)).Return(nil, domain.ErrProductNotFound)
            },
            wantErr: domain.ErrProductNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc, repo := newProductService(t)
            tt.setup(repo)

            p, err := svc.Update(context.Background(), tt.actor, 10, service.UpdateProductInput{Name: &newName})

            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
                return
            }
            require.NoError(t, err)
            require.Equal(t, "New", p.Name)
            require.Equal(t, fixedNow, p.UpdatedAt)
        })
    }
}
```

`package service_test` (external test package) — faqat public API test qilinadi, import cycle bo'lmaydi.

## 2. Handler test — httptest

Handler `*service.ProductService` (struct) oladi — mock qilib bo'lmaydi. Ikki yo'l:
- **(a)** Handler interfeys olsin: `type ProductService interface { Create(...); Get(...) }` handler paketida e'lon qilinadi → mockery bilan mock. **Tavsiya.**
- **(b)** Real service + mock repository (biroz kengroq test).

(a) bilan:

```go
package handler_test

func TestProductHandler_Get(t *testing.T) {
    svc := mocks.NewProductService(t) // handler paketidagi interfeys mock'i
    svc.EXPECT().Get(mock.Anything, int64(7)).
        Return(&domain.Product{ID: 7, Name: "Phone", Status: domain.ProductDraft}, nil)

    v, _ := validator.New()
    h := handler.NewProductHandler(svc, v)

    r := chi.NewRouter()
    r.Get("/products/{id}", h.Get) // URLParam ishlashi uchun router kerak

    req := httptest.NewRequest(http.MethodGet, "/products/7", nil)
    rec := httptest.NewRecorder()
    r.ServeHTTP(rec, req)

    require.Equal(t, http.StatusOK, rec.Code)
    var body dto.ProductResponse
    require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
    require.Equal(t, "Phone", body.Name)
}

func TestProductHandler_Create_ValidationError(t *testing.T) {
    svc := mocks.NewProductService(t) // hech narsa kutilmaydi — chaqirilsa test yiqiladi
    v, _ := validator.New()
    h := handler.NewProductHandler(svc, v)

    body := strings.NewReader(`{"name":"", "price": -1}`)
    req := httptest.NewRequest(http.MethodPost, "/products", body)
    req = req.WithContext(middleware.WithUser(req.Context(), &domain.User{ID: 1}))
    rec := httptest.NewRecorder()
    h.Create(rec, req)

    require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
    require.Contains(t, rec.Body.String(), `"field":"name"`)
}
```

## 3. Integration test — repository (testcontainers)

`internal/repository/postgres/main_test.go` — bitta container barcha testlar uchun:

```go
//go:build integration

package postgres_test

import (
    "context"
    "database/sql"
    "os"
    "testing"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/pressly/goose/v3"
    _ "github.com/jackc/pgx/v5/stdlib"
    "github.com/testcontainers/testcontainers-go"
    tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
    ctx := context.Background()

    ctr, err := tcpg.Run(ctx, "postgres:16-alpine",
        tcpg.WithDatabase("test"), tcpg.WithUsername("test"), tcpg.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).WithStartupTimeout(30*time.Second)),
    )
    if err != nil {
        panic(err)
    }
    dsn, _ := ctr.ConnectionString(ctx, "sslmode=disable")

    // migratsiya: goose *sql.DB bilan ishlaydi
    sqlDB, err := sql.Open("pgx", dsn)
    if err != nil {
        panic(err)
    }
    if err := goose.SetDialect("postgres"); err != nil {
        panic(err)
    }
    if err := goose.Up(sqlDB, "../../../migrations"); err != nil {
        panic(err)
    }
    sqlDB.Close()

    testPool, _ = pgxpool.New(ctx, dsn)

    code := m.Run()

    testPool.Close()
    _ = ctr.Terminate(ctx)
    os.Exit(code)
}

// Har test oldidan jadvalni tozalash
func truncate(t *testing.T) {
    t.Helper()
    _, err := testPool.Exec(context.Background(), `TRUNCATE products, users RESTART IDENTITY CASCADE`)
    require.NoError(t, err)
}
```

`product_repository_test.go`:

```go
//go:build integration

func TestProductRepository_CreateAndGet(t *testing.T) {
    truncate(t)
    ctx := context.Background()
    repo := postgres.NewProductRepository(testPool)
    // users FK uchun avval user kerak — helper yozing: createTestUser(t)

    p := &domain.Product{OwnerID: createTestUser(t), Name: "Phone", Price: 9.99, Status: domain.ProductDraft,
        CreatedAt: time.Now(), UpdatedAt: time.Now()}
    require.NoError(t, repo.Create(ctx, p))
    require.NotZero(t, p.ID)

    got, err := repo.GetByID(ctx, p.ID)
    require.NoError(t, err)
    require.Equal(t, "Phone", got.Name)

    _, err = repo.GetByID(ctx, 9999)
    require.ErrorIs(t, err, domain.ErrProductNotFound)
}
```

```bash
go test ./...                              # unit + handler (tez)
go test -tags=integration ./...            # + integration (Docker kerak)
go test -race -coverprofile=cover.out ./... && go tool cover -html=cover.out -o coverage.html
```

## Qoidalar

1. Test nomi: `Test<Type>_<Method>_<Holat>` — `TestProductService_Update_Forbidden`.
2. `require` (xato bo'lsa to'xtaydi) — `assert` (davom etadi). Ko'pincha `require`.
3. Har test **mustaqil**: o'z mock'i, o'z `truncate`. Global holat yo'q.
4. `t.Parallel()` — mustaqil unit testlarda.
5. `t.Helper()` — helper funksiyalarda; xato qatori to'g'ri ko'rsatiladi.
6. Vaqt, random — interfeys orqali → deterministik.
7. Coverage maqsad: service 80%+, handler asosiy yo'llar, repository har metod bitta happy + bitta not-found.
8. Mock'ni `mocks/` papkasida saqlang, `go generate` yoki `make mocks` bilan yangilang; commit qiling.
