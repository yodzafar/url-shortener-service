# 16 — Go HTTP frameworklar: taqqoslash va ulash

## Taqqoslash

| | `net/http` (std) | `chi` | `gin` | `echo` | `fiber` |
|---|---|---|---|---|---|
| Asos | std | std `http.Handler` | o'z `gin.Context` | o'z `echo.Context` | `fasthttp` (std emas) |
| Handler imzosi | `(w, r)` | `(w, r)` | `(c *gin.Context)` | `(c echo.Context) error` | `(c *fiber.Ctx) error` |
| Std middleware mos | ✅ | ✅ | adapter kerak | adapter kerak | ❌ (fasthttp) |
| Tezlik | yaxshi | yaxshi | yaxshi | yaxshi | eng tez (lekin farq real loyihada sezilmaydi) |
| Mashhurlik | — | yuqori | eng yuqori | yuqori | o'sib bormoqda |
| Path param | `r.PathValue("id")` (1.22+) | `chi.URLParam(r,"id")` | `c.Param("id")` | `c.Param("id")` | `c.Params("id")` |
| Bind+validate | qo'lda | qo'lda | `c.ShouldBindJSON` (validator ichida) | `c.Bind` + `c.Validate` | `c.BodyParser` |

## Tavsiya

- **O'rganish uchun: `chi`** — std `http.Handler` ustida; o'rgangan hamma narsa (middleware, httptest) `net/http`ga to'g'ri keladi. Framework almashtirsangiz faqat `handler` va `router.go` o'zgaradi.
- **Ishga kirish uchun: `gin`** — eng ko'p vakansiya. chi'dan keyin 1 kunda o'rganasiz.
- `fiber` — std bilan mos emas (`http.Handler` yo'q), ekosistema kichikroq. Std'ni o'rganmasdan boshlamang.
- `echo` — gin'ga o'xshash, handler `error` qaytaradi (markaziy error handler qulay).

**Clean Architecture'da framework faqat `transport/http`da.** Service, domain, repository — framework'ni bilmaydi. Shuning uchun almashtirish arzon.

## Bir xil service — turli frameworklar

Service: `svc.Get(ctx, id) (*domain.Product, error)`.

### net/http (Go 1.22+ method routing)

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /api/v1/products/{id}", h.Get)
mux.Handle("POST /api/v1/products", authMW(http.HandlerFunc(h.Create)))

func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
    ...
}
```

Kichik servis uchun hech qanday framework kerak emas. Kamchilik: route group, middleware zanjiri qo'lda.

### chi — [09-handler-router.md](09-handler-router.md)da to'liq.

### gin

```go
import "github.com/gin-gonic/gin"

type ProductHandler struct{ svc ProductService }

func (h *ProductHandler) Get(c *gin.Context) {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "invalid id"}})
        return
    }
    p, err := h.svc.Get(c.Request.Context(), id)
    if err != nil {
        response.GinFromError(c, err) // FromError'ning gin varianti
        return
    }
    c.JSON(http.StatusOK, dto.FromProduct(p))
}

func (h *ProductHandler) Create(c *gin.Context) {
    var req dto.CreateProductRequest
    if err := c.ShouldBindJSON(&req); err != nil { // binding:"required" teglari bilan validator ichida
        c.JSON(http.StatusBadRequest, ...)
        return
    }
    ...
}

// router
r := gin.New()
r.Use(gin.Recovery(), middleware.GinLogger(log))
v1 := r.Group("/api/v1")
{
    v1.POST("/auth/login", auth.Login)
    v1.GET("/products/:id", product.Get)
    protected := v1.Group("", authMW.Authenticate())
    protected.POST("/products", product.Create)
    admin := protected.Group("/admin", authMW.RequireRole(domain.RoleAdmin))
    admin.GET("/users", auth.ListUsers)
}
```

gin middleware:

```go
func (a *Auth) Authenticate() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
        user, err := a.auth.UserFromToken(c.Request.Context(), token)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, ...)
            return
        }
        c.Set("user", user)       // gin context
        c.Next()
    }
}
// handler'da: user := c.MustGet("user").(*domain.User)
```

gin'da o'z validatorni ulash: `binding.Validator = &customValidator{}` yoki `c.ShouldBindJSON` o'rniga `json.Decode` + o'z `validator.Validate` (Clean Arch uchun tavsiya — bitta validator hamma joyda).

### echo

```go
import "github.com/labstack/echo/v4"

func (h *ProductHandler) Get(c echo.Context) error {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
    }
    p, err := h.svc.Get(c.Request().Context(), id)
    if err != nil {
        return err // markaziy HTTPErrorHandler domain error'ni statusga o'giradi
    }
    return c.JSON(http.StatusOK, dto.FromProduct(p))
}

e := echo.New()
e.HTTPErrorHandler = response.EchoErrorHandler // errMap bilan
e.Use(echomw.Recover(), echomw.RequestID())
v1 := e.Group("/api/v1")
v1.GET("/products/:id", product.Get)
protected := v1.Group("", authMW.Authenticate)
protected.POST("/products", product.Create)
```

Echo'ning "handler `error` qaytaradi" modeli — har handler'da `response.FromError` chaqirmaysiz.

### fiber (v2)

```go
import "github.com/gofiber/fiber/v2"

func (h *ProductHandler) Get(c *fiber.Ctx) error {
    id, err := c.ParamsInt("id")
    if err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "invalid id")
    }
    p, err := h.svc.Get(c.UserContext(), int64(id)) // c.Context() — fasthttp ctx, UserContext() — std
    if err != nil {
        return err
    }
    return c.Status(fiber.StatusOK).JSON(dto.FromProduct(p))
}

app := fiber.New(fiber.Config{ErrorHandler: response.FiberErrorHandler})
app.Use(recover.New())
v1 := app.Group("/api/v1")
v1.Get("/products/:id", product.Get)
```

Fiber'da `context.Context` — `c.UserContext()`. `httptest` ishlamaydi — `app.Test(req)` ishlatiladi.

## Framework almashtirganda nima o'zgaradi

```
o'zgaradi:   transport/http/handler/*, router.go, middleware/*, response/* (framework-specific qismi)
o'zgarmaydi: domain, service, repository, dto (struct'lar), pkg/validator, pkg/jwt, testlar (service)
```

Shu sababli handler paketi service'ni **interfeys** orqali olsa — handler testi ham framework'dan mustaqil bo'ladi.

## Qo'shimcha foydali kutubxonalar

| Vazifa | Kutubxona |
|--------|-----------|
| CORS | `go-chi/cors`, `gin-contrib/cors` |
| Rate limit | `go-chi/httprate`, `ulule/limiter` |
| Query builder | `Masterminds/squirrel` |
| SQL → Go codegen | `sqlc-dev/sqlc` |
| ORM | `gorm.io/gorm`, `uptrace/bun`, `ent` |
| gRPC | `google.golang.org/grpc` + `buf` |
| Background job | `hibiken/asynq` (Redis), `riverqueue/river` (Postgres) |
| Metrics | `prometheus/client_golang` |
| Tracing | `go.opentelemetry.io/otel` |
| UUID | `google/uuid` |
| Decimal (pul) | `shopspring/decimal` |
| Hot reload | `air-verse/air` |
