# 09 — Handler va Router (chi)

Handler yupqa: **decode → validate → service → response**. Biznes mantiq yo'q.

## `internal/transport/http/handler/product_handler.go`

```go
package handler

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/go-chi/chi/v5"

    "github.com/yodzafar/myservice/internal/domain"
    "github.com/yodzafar/myservice/internal/service"
    "github.com/yodzafar/myservice/internal/transport/http/dto"
    "github.com/yodzafar/myservice/internal/transport/http/middleware"
    "github.com/yodzafar/myservice/internal/transport/http/response"
    "github.com/yodzafar/myservice/pkg/validator"
)

type ProductHandler struct {
    svc       *service.ProductService
    validator *validator.Validator
}

func NewProductHandler(svc *service.ProductService, v *validator.Validator) *ProductHandler {
    return &ProductHandler{svc: svc, validator: v}
}

// Create  POST /api/v1/products
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req dto.CreateProductRequest
    if err := decodeJSON(r, &req); err != nil {
        response.Error(w, http.StatusBadRequest, "invalid json body")
        return
    }
    if ferrs := h.validator.Validate(req); ferrs != nil {
        response.ValidationError(w, ferrs)
        return
    }

    user := middleware.UserFromContext(r.Context()) // auth middleware qo'ygan
    p, err := h.svc.Create(r.Context(), req.ToInput(user.ID))
    if err != nil {
        response.FromError(w, r, err) // domain error → status
        return
    }
    response.JSON(w, http.StatusCreated, dto.FromProduct(p))
}

// Get  GET /api/v1/products/{id}
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        response.Error(w, http.StatusBadRequest, "invalid id")
        return
    }
    p, err := h.svc.Get(r.Context(), id)
    if err != nil {
        response.FromError(w, r, err)
        return
    }
    response.JSON(w, http.StatusOK, dto.FromProduct(p))
}

// Update  PATCH /api/v1/products/{id}
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        response.Error(w, http.StatusBadRequest, "invalid id")
        return
    }
    var req dto.UpdateProductRequest
    if err := decodeJSON(r, &req); err != nil {
        response.Error(w, http.StatusBadRequest, "invalid json body")
        return
    }
    if ferrs := h.validator.Validate(req); ferrs != nil {
        response.ValidationError(w, ferrs)
        return
    }
    user := middleware.UserFromContext(r.Context())
    p, err := h.svc.Update(r.Context(), user, id, req.ToInput())
    if err != nil {
        response.FromError(w, r, err)
        return
    }
    response.JSON(w, http.StatusOK, dto.FromProduct(p))
}

// List  GET /api/v1/products?status=published&page=1&per_page=20
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    query := dto.ListProductsQuery{
        Status:  q.Get("status"),
        Page:    atoiDefault(q.Get("page"), 1),
        PerPage: atoiDefault(q.Get("per_page"), 20),
    }
    if ferrs := h.validator.Validate(query); ferrs != nil {
        response.ValidationError(w, ferrs)
        return
    }
    in := service.ListProductsInput{Page: query.Page, PerPage: query.PerPage}
    if query.Status != "" {
        st := domain.ProductStatus(query.Status)
        in.Status = &st
    }
    items, total, err := h.svc.List(r.Context(), in)
    if err != nil {
        response.FromError(w, r, err)
        return
    }
    response.JSON(w, http.StatusOK, dto.ListProductsResponse{
        Items: dto.FromProducts(items), Total: total, Page: query.Page, PerPage: query.PerPage,
    })
}

// Delete  DELETE /api/v1/products/{id}
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        response.Error(w, http.StatusBadRequest, "invalid id")
        return
    }
    user := middleware.UserFromContext(r.Context())
    if err := h.svc.Delete(r.Context(), user, id); err != nil {
        response.FromError(w, r, err)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

// ---------- helpers (handler/helpers.go) ----------

func decodeJSON(r *http.Request, dst any) error {
    r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1MB limit
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields() // noma'lum field → xato (ixtiyoriy, lekin foydali)
    return dec.Decode(dst)
}

func atoiDefault(s string, def int) int {
    if s == "" {
        return def
    }
    n, err := strconv.Atoi(s)
    if err != nil {
        return def
    }
    return n
}
```

## `internal/transport/http/router.go`

```go
package http

import (
    "log/slog"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    chimw "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/cors"
    httpSwagger "github.com/swaggo/http-swagger/v2"

    "github.com/yodzafar/myservice/internal/domain"
    "github.com/yodzafar/myservice/internal/transport/http/handler"
    "github.com/yodzafar/myservice/internal/transport/http/middleware"
)

type RouterDeps struct {
    Auth     *handler.AuthHandler
    Product  *handler.ProductHandler
    AuthMW   *middleware.Auth
    Logger   *slog.Logger
}

func NewRouter(d RouterDeps) http.Handler {
    r := chi.NewRouter()

    // Global middleware — tartib muhim
    r.Use(chimw.RequestID)
    r.Use(chimw.RealIP)
    r.Use(middleware.Logger(d.Logger))
    r.Use(chimw.Recoverer)
    r.Use(chimw.Timeout(30 * time.Second))
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins:   []string{"http://localhost:3000"},
        AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Accept-Language"},
        AllowCredentials: true,
    }))

    r.Get("/health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
    r.Get("/swagger/*", httpSwagger.WrapHandler)

    r.Route("/api/v1", func(r chi.Router) {
        // public
        r.Post("/auth/register", d.Auth.Register)
        r.Post("/auth/login", d.Auth.Login)
        r.Post("/auth/refresh", d.Auth.Refresh)

        r.Get("/products", d.Product.List)
        r.Get("/products/{id}", d.Product.Get)

        // faqat login qilganlar
        r.Group(func(r chi.Router) {
            r.Use(d.AuthMW.Authenticate)

            r.Get("/auth/me", d.Auth.Me)
            r.Post("/products", d.Product.Create)
            r.Patch("/products/{id}", d.Product.Update)
            r.Delete("/products/{id}", d.Product.Delete)
            r.Post("/products/{id}/publish", d.Product.Publish)
        })

        // faqat admin
        r.Group(func(r chi.Router) {
            r.Use(d.AuthMW.Authenticate, d.AuthMW.RequireRole(domain.RoleAdmin))

            r.Get("/admin/users", d.Auth.ListUsers)
        })
    })

    return r
}
```

Router'da `d.Product.Publish`, `d.Auth.ListUsers` kabi handler'lar yuqoridagi namunalar asosida yoziladi.

## `internal/transport/http/server.go`

```go
package http

import (
    "net/http"
    "strconv"
    "time"
)

func NewServer(port int, h http.Handler, readTimeout, writeTimeout time.Duration) *http.Server {
    return &http.Server{
        Addr:              ":" + strconv.Itoa(port),
        Handler:           h,
        ReadTimeout:       readTimeout,
        ReadHeaderTimeout: 5 * time.Second,
        WriteTimeout:      writeTimeout,
        IdleTimeout:       60 * time.Second,
    }
}
```

Timeout'lar **shart** — bo'lmasa sekin klient serverni osib qo'yadi.

## REST konvensiyalari

| Amal | Method | Path | Status |
|------|--------|------|--------|
| Yaratish | POST | `/products` | 201 + obyekt |
| O'qish | GET | `/products/{id}` | 200 |
| Ro'yxat | GET | `/products?page=1` | 200 |
| Qisman yangilash | PATCH | `/products/{id}` | 200 |
| To'liq almashtirish | PUT | `/products/{id}` | 200 |
| O'chirish | DELETE | `/products/{id}` | 204 (body yo'q) |
| Amal (fe'l) | POST | `/products/{id}/publish` | 200 |

- Path'da ko'plik (`products`), kichik harf, `-` ajratgich.
- Versiya prefiksda: `/api/v1`.
- Filter/pagination — query param. Body faqat POST/PATCH/PUT.

## Qoidalar

1. Handler'da `if err != nil { response.FromError(...); return }` — **har doim `return`**. Unutish = ikki marta yozish (`superfluous WriteHeader`).
2. `r.Context()` ni service'ga uzating — klient uzilsa DB query bekor bo'ladi.
3. Body hajmini cheklang (`MaxBytesReader`).
4. Handler struct'da faqat service va validator. Repository, DB — yo'q.
5. Handler paketida chi importi faqat `URLParam` uchun. Framework almashtirsangiz shu joy o'zgaradi — [16-frameworks.md](16-frameworks.md).
