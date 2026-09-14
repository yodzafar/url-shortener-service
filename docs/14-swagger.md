# 14 — Swagger (OpenAPI) — swaggo

`swag` handler kommentlaridan `swagger.json` generatsiya qiladi; `http-swagger` uni UI sifatida ko'rsatadi.

```bash
go install github.com/swaggo/swag/cmd/swag@latest
go get github.com/swaggo/swag
go get github.com/swaggo/http-swagger/v2
```

## 1. Umumiy annotatsiya — `cmd/api/main.go`

```go
// @title           My Service API
// @version         1.0
// @description     Product management service.
// @host            localhost:8080
// @BasePath        /api/v1
// @schemes         http https

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                "Bearer <access_token>" formatida
package main
```

## 2. Handler annotatsiyasi

```go
// Create godoc
// @Summary      Mahsulot yaratish
// @Description  Login qilgan foydalanuvchi yangi mahsulot yaratadi (status: draft)
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        request body     dto.CreateProductRequest true "Mahsulot ma'lumotlari"
// @Success      201     {object} dto.ProductResponse
// @Failure      400     {object} response.ErrorResponse
// @Failure      401     {object} response.ErrorResponse
// @Failure      422     {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /products [post]
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) { ... }

// Get godoc
// @Summary   Mahsulotni olish
// @Tags      products
// @Produce   json
// @Param     id   path      int  true  "Product ID"
// @Success   200  {object}  dto.ProductResponse
// @Failure   404  {object}  response.ErrorResponse
// @Router    /products/{id} [get]
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) { ... }

// List godoc
// @Summary   Mahsulotlar ro'yxati
// @Tags      products
// @Produce   json
// @Param     status    query     string  false  "draft | published"
// @Param     page      query     int     false  "Sahifa"  default(1)
// @Param     per_page  query     int     false  "Sahifada nechta"  default(20)  maximum(100)
// @Success   200  {object}  dto.ListProductsResponse
// @Router    /products [get]
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) { ... }

// Login godoc
// @Summary   Login
// @Tags      auth
// @Accept    json
// @Produce   json
// @Param     request  body      dto.LoginRequest  true  "Credentials"
// @Success   200      {object}  dto.TokenResponse
// @Failure   401      {object}  response.ErrorResponse
// @Router    /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) { ... }
```

DTO'larga example va tavsif:

```go
type CreateProductRequest struct {
    Name  string  `json:"name"  validate:"required,min=2" example:"iPhone 15"`
    Price float64 `json:"price" validate:"gte=0"          example:"999.99"`
}
```

## 3. Generatsiya

```bash
swag init -g cmd/api/main.go -o api/swagger --parseDependency --parseInternal
```

- `-g` — umumiy annotatsiya qayerda
- `-o api/swagger` — chiqish papkasi (`docs.go`, `swagger.json`, `swagger.yaml`)
- `--parseDependency --parseInternal` — `internal/` va tashqi paket tiplarini ham o'qisin

Makefile'ga: `swagger: ; swag init -g cmd/api/main.go -o api/swagger --parseDependency --parseInternal`

Har annotatsiya o'zgarganda qayta ishga tushiring. `api/swagger/` commit qilinadi (yoki CI'da generatsiya).

## 4. Router'ga ulash

```go
import (
    httpSwagger "github.com/swaggo/http-swagger/v2"

    _ "github.com/yodzafar/myservice/api/swagger" // generatsiya qilingan docs.go — side-effect import
)

r.Get("/swagger/*", httpSwagger.Handler(
    httpSwagger.URL("/swagger/doc.json"),
))
```

Ochish: `http://localhost:8080/swagger/index.html`

Prod'da swagger'ni yopish: `if cfg.IsLocal() { r.Get("/swagger/*", ...) }`.

## 5. JWT bilan test qilish

Swagger UI'da **Authorize** tugmasi → `Bearer eyJhbGci...` kiriting. `@Security BearerAuth` yozilgan endpointlar header'ni avtomatik yuboradi.

## Boshqa frameworklar

| Framework | Paket | Route |
|-----------|-------|-------|
| chi / net/http | `swaggo/http-swagger/v2` | `r.Get("/swagger/*", httpSwagger.WrapHandler)` |
| gin | `swaggo/gin-swagger` + `swaggo/files` | `r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))` |
| echo | `swaggo/echo-swagger` | `e.GET("/swagger/*", echoSwagger.WrapHandler)` |
| fiber | `gofiber/swagger` | `app.Get("/swagger/*", swagger.HandlerDefault)` |

## Alternativa: spec-first

Avval `openapi.yaml` yozib, undan kod generatsiya: `oapi-codegen` (`github.com/oapi-codegen/oapi-codegen/v2`). Katta jamoada frontend bilan kontrakt oldindan kelishilganda qulay. O'rganish uchun swaggo (code-first) osonroq.
