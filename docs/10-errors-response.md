# 10 — Xatolar va yagona Response formati

## Response formati

Muvaffaqiyat — to'g'ridan-to'g'ri obyekt (yoki `{items, total}`). Xato — har doim bir xil shakl:

```json
{
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "message": "product not found",
    "fields": [ { "field": "price", "message": "price must be >= 0" } ]
  }
}
```

`code` — mashina uchun (frontend switch qiladi), `message` — odam uchun, `fields` — faqat validation'da.

## `internal/transport/http/response/response.go`

```go
package response

import (
    "encoding/json"
    "log/slog"
    "net/http"

    "github.com/yodzafar/myservice/pkg/validator"
)

type ErrorBody struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Fields  []validator.FieldError `json:"fields,omitempty"`
}

type ErrorResponse struct {
    Error ErrorBody `json:"error"`
}

func JSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(status)
    if v == nil {
        return
    }
    if err := json.NewEncoder(w).Encode(v); err != nil {
        slog.Error("write response", "err", err)
    }
}

func Error(w http.ResponseWriter, status int, msg string) {
    JSON(w, status, ErrorResponse{Error: ErrorBody{Code: codeFromStatus(status), Message: msg}})
}

func ValidationError(w http.ResponseWriter, fields []validator.FieldError) {
    JSON(w, http.StatusUnprocessableEntity, ErrorResponse{Error: ErrorBody{
        Code: "VALIDATION_ERROR", Message: "validation failed", Fields: fields,
    }})
}

func codeFromStatus(s int) string {
    switch s {
    case http.StatusBadRequest:   return "BAD_REQUEST"
    case http.StatusUnauthorized: return "UNAUTHORIZED"
    case http.StatusForbidden:    return "FORBIDDEN"
    case http.StatusNotFound:     return "NOT_FOUND"
    case http.StatusConflict:     return "CONFLICT"
    default:                      return "INTERNAL_ERROR"
    }
}
```

## `internal/transport/http/response/errors.go` — domain error → HTTP

```go
package response

import (
    "context"
    "errors"
    "log/slog"
    "net/http"

    "github.com/yodzafar/myservice/internal/domain"
)

type mapping struct {
    status int
    code   string
}

// Bitta jadval — yangi domain xato qo'shsangiz shu yerga qo'shasiz
var errMap = map[error]mapping{
    domain.ErrNotFound:           {http.StatusNotFound, "NOT_FOUND"},
    domain.ErrProductNotFound:    {http.StatusNotFound, "PRODUCT_NOT_FOUND"},
    domain.ErrUserNotFound:       {http.StatusNotFound, "USER_NOT_FOUND"},
    domain.ErrEmailTaken:         {http.StatusConflict, "EMAIL_TAKEN"},
    domain.ErrAlreadyPublished:   {http.StatusConflict, "ALREADY_PUBLISHED"},
    domain.ErrInvalidCredentials: {http.StatusUnauthorized, "INVALID_CREDENTIALS"},
    domain.ErrInvalidToken:       {http.StatusUnauthorized, "INVALID_TOKEN"},
    domain.ErrForbidden:          {http.StatusForbidden, "FORBIDDEN"},
    domain.ErrInvalidPrice:       {http.StatusBadRequest, "INVALID_PRICE"},
    domain.ErrInvalidProductName: {http.StatusBadRequest, "INVALID_NAME"},
}

// FromError — har qanday xatoni to'g'ri HTTP javobga aylantiradi.
func FromError(w http.ResponseWriter, r *http.Request, err error) {
    for target, m := range errMap {
        if errors.Is(err, target) {
            JSON(w, m.status, ErrorResponse{Error: ErrorBody{Code: m.code, Message: target.Error()}})
            return
        }
    }

    // Klient uzildi / timeout
    if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
        Error(w, http.StatusRequestTimeout, "request cancelled")
        return
    }

    // Noma'lum xato — 500. Ichki xabar klientga CHIQMAYDI, faqat log
    slog.ErrorContext(r.Context(), "unhandled error", "err", err, "path", r.URL.Path)
    Error(w, http.StatusInternalServerError, "internal server error")
}
```

`errors.Is` wrap qilingan xatolarni ham topadi: `fmt.Errorf("create product: %w", domain.ErrInvalidPrice)` → `400`.

## Qo'shimcha ma'lumotli xato (custom error type)

Sentinel yetmasa (masalan, qaysi ID topilmadi):

```go
// domain/errors.go
type NotFoundError struct {
    Entity string
    ID     int64
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s %d not found", e.Entity, e.ID)
}

func (e *NotFoundError) Is(target error) bool { return target == ErrNotFound } // errors.Is(err, ErrNotFound) ishlaydi

// repository
return nil, &domain.NotFoundError{Entity: "product", ID: id}

// transport
var nf *domain.NotFoundError
if errors.As(err, &nf) {
    JSON(w, 404, ErrorResponse{Error: ErrorBody{Code: "NOT_FOUND", Message: nf.Error()}})
}
```

## Status kodlar — qachon qaysi

| Status | Qachon |
|--------|--------|
| 400 | JSON buzuq, param noto'g'ri turda, biznes qoida buzilgan (invalid price) |
| 401 | Token yo'q / noto'g'ri / muddati o'tgan |
| 403 | Token to'g'ri, lekin ruxsat yo'q (rol, owner emas) |
| 404 | Resurs topilmadi |
| 409 | Konflikt: unique (email band), holat (allaqachon published) |
| 422 | Validation xato (field'lar bilan) |
| 429 | Rate limit |
| 500 | Kutilmagan xato — hech qachon ichki xabarni chiqarmang |

## Qoidalar

1. Xato faqat **bir joyda** log qilinadi — `FromError`da (500 uchun). Service/repository log qilmaydi, wrap qiladi.
2. 500 javobda ichki xabar (`pq: relation does not exist`) chiqmasin — xavfsizlik.
3. Xato zanjirini saqlang: `%w`. `%v` — zanjir uziladi, `errors.Is` ishlamaydi.
4. `errMap` jadvali — yangi xato qo'shganda handler'larga tegmaysiz.
