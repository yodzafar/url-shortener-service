# 08 — DTO, Mapping, Validation, Custom validation, Tarjimalar

## To'rt xil struct — nega?

| Struct | Qayerda | Teglar | Vazifa |
|--------|---------|--------|--------|
| `dto.CreateProductRequest` | transport/http/dto | `json`, `validate` | HTTP body shakli |
| `service.CreateProductInput` | service | yo'q | use case kirishi |
| `domain.Product` | domain | yo'q | biznes obyekt |
| `productRow` (ixtiyoriy) | repository | `db` | DB qatori — faqat domain ≠ jadval bo'lsa; 1:1 bo'lsa `domain.Product`ga `db` teg |

API o'zgarsa DTO o'zgaradi, domain tegilmaydi. DB o'zgarsa `productRow` o'zgaradi.

## `internal/transport/http/dto/product_dto.go`

```go
package dto

import (
    "time"

    "github.com/yodzafar/myservice/internal/domain"
    "github.com/yodzafar/myservice/internal/service"
)

// ---------- Request ----------

type CreateProductRequest struct {
    Name  string  `json:"name"  validate:"required,min=2,max=255"`
    Price float64 `json:"price" validate:"required,gte=0"`
}

type UpdateProductRequest struct {
    Name  *string  `json:"name,omitempty"  validate:"omitempty,min=2,max=255"`
    Price *float64 `json:"price,omitempty" validate:"omitempty,gte=0"`
}

type ListProductsQuery struct {
    Status  string `validate:"omitempty,oneof=draft published"`
    Page    int    `validate:"omitempty,min=1"`
    PerPage int    `validate:"omitempty,min=1,max=100"`
}

// ---------- Response ----------

type ProductResponse struct {
    ID        int64     `json:"id"`
    OwnerID   int64     `json:"owner_id"`
    Name      string    `json:"name"`
    Price     float64   `json:"price"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}

type ListProductsResponse struct {
    Items   []ProductResponse `json:"items"`
    Total   int               `json:"total"`
    Page    int               `json:"page"`
    PerPage int               `json:"per_page"`
}

// ---------- Mapping ----------

func (r CreateProductRequest) ToInput(ownerID int64) service.CreateProductInput {
    return service.CreateProductInput{OwnerID: ownerID, Name: r.Name, Price: r.Price}
}

func (r UpdateProductRequest) ToInput() service.UpdateProductInput {
    return service.UpdateProductInput{Name: r.Name, Price: r.Price}
}

func FromProduct(p *domain.Product) ProductResponse {
    return ProductResponse{
        ID: p.ID, OwnerID: p.OwnerID, Name: p.Name, Price: p.Price,
        Status: string(p.Status), CreatedAt: p.CreatedAt,
    }
}

func FromProducts(ps []domain.Product) []ProductResponse {
    out := make([]ProductResponse, 0, len(ps)) // bo'sh bo'lsa null emas, [] qaytadi
    for i := range ps {
        out = append(out, FromProduct(&ps[i]))
    }
    return out
}
```

Mapping qoidalari:
- **Alohida fayl**: `dto/mapper.go` (yoki `product_mapper.go`) — `product_dto.go`da faqat struct'lar va teglar qoladi. Repository tomonida ham xuddi shunday: `postgres/mapper.go` ([06-repository.md](06-repository.md)).
- Nomlash: `ToInput()` (request → service input), `FromProduct()` (domain → response). `toDto` emas — Go'da `DTO` bosh harf bilan, lekin `From/To` aniqroq.
- Slice uchun `FromProducts` yozish o'rniga generic `slices.Map(products, FromProduct)` ([06-repository.md](06-repository.md)dagi yordamchi).
- Oddiy funksiyalar. Reflection kutubxona (`copier`, `mapstructure`) kerak emas — aniq va tez. 30+ entity va bir xil field nomlari bo'lsa `goverter` (compile-time generatsiya) ko'ring.
- Response'da ichki/maxfiy field (`password_hash`) hech qachon chiqmasin — DTO shu uchun.
- Ko'p DTO bo'lsa `dto/mapper.go`ga chiqaring.

## Validator — `pkg/validator/validator.go`

```go
package validator

import (
    "errors"
    "reflect"
    "regexp"
    "strings"
    "time"

    "github.com/go-playground/locales/en"
    ut "github.com/go-playground/universal-translator"
    "github.com/go-playground/validator/v10"
    en_translations "github.com/go-playground/validator/v10/translations/en"
)

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Validator struct {
    v     *validator.Validate
    trans ut.Translator
}

// FieldError — handler'ga qaytariladigan tayyor xato
type FieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

func New() (*Validator, error) {
    v := validator.New(validator.WithRequiredStructEnabled())

    // Xato xabarida struct field emas, json nomi chiqsin: "Name" → "name"
    v.RegisterTagNameFunc(func(fld reflect.StructField) string {
        name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
        if name == "-" || name == "" {
            return fld.Name
        }
        return name
    })

    // --- Custom rule'lar ---
    if err := v.RegisterValidation("slug", validateSlug); err != nil {
        return nil, err
    }
    if err := v.RegisterValidation("future", validateFuture); err != nil {
        return nil, err
    }
    if err := v.RegisterValidation("strong_password", validateStrongPassword); err != nil {
        return nil, err
    }

    // --- Tarjima (ingliz) ---
    enLocale := en.New()
    uni := ut.New(enLocale, enLocale)
    trans, _ := uni.GetTranslator("en")
    if err := en_translations.RegisterDefaultTranslations(v, trans); err != nil {
        return nil, err
    }
    registerTranslation(v, trans, "slug", "{0} must be a slug: lowercase letters, digits and '-'")
    registerTranslation(v, trans, "future", "{0} must be a date in the future")
    registerTranslation(v, trans, "strong_password", "{0} must contain upper, lower case letters and a digit")

    return &Validator{v: v, trans: trans}, nil
}

// Validate — nil qaytarsa hammasi to'g'ri
func (vd *Validator) Validate(s any) []FieldError {
    err := vd.v.Struct(s)
    if err == nil {
        return nil
    }
    var verrs validator.ValidationErrors
    if !errors.As(err, &verrs) {
        return []FieldError{{Message: err.Error()}}
    }
    out := make([]FieldError, 0, len(verrs))
    for _, fe := range verrs {
        out = append(out, FieldError{Field: fe.Field(), Message: fe.Translate(vd.trans)})
    }
    return out
}

// ---------- custom rule funksiyalari ----------

func validateSlug(fl validator.FieldLevel) bool {
    return slugRe.MatchString(fl.Field().String())
}

func validateFuture(fl validator.FieldLevel) bool {
    t, ok := fl.Field().Interface().(time.Time)
    return ok && t.After(time.Now())
}

func validateStrongPassword(fl validator.FieldLevel) bool {
    s := fl.Field().String()
    var upper, lower, digit bool
    for _, c := range s {
        switch {
        case c >= 'A' && c <= 'Z': upper = true
        case c >= 'a' && c <= 'z': lower = true
        case c >= '0' && c <= '9': digit = true
        }
    }
    return upper && lower && digit
}

func registerTranslation(v *validator.Validate, trans ut.Translator, tag, msg string) {
    _ = v.RegisterTranslation(tag, trans,
        func(ut ut.Translator) error { return ut.Add(tag, msg, true) },
        func(ut ut.Translator, fe validator.FieldError) string {
            t, _ := ut.T(tag, fe.Field())
            return t
        },
    )
}
```

Ishlatish: `validate:"required,strong_password,min=8"`.

## Custom rule'ga parametr berish

`validate:"max_words=5"`:

```go
v.RegisterValidation("max_words", func(fl validator.FieldLevel) bool {
    n, err := strconv.Atoi(fl.Param()) // "5"
    if err != nil { return false }
    return len(strings.Fields(fl.Field().String())) <= n
})
```

## Struct-level validation (ikki field bog'liq)

```go
v.RegisterStructValidation(func(sl validator.StructLevel) {
    r := sl.Current().Interface().(DateRangeRequest)
    if r.To.Before(r.From) {
        sl.ReportError(r.To, "to", "To", "after_from", "")
    }
}, DateRangeRequest{})
```

## Ko'p tilli validation xabarlari

`validator/v10/translations/` ichida `en`, `ru`, `tr`, `zh`, ... bor; `uz` yo'q — o'zingiz `registerTranslation` bilan har bir teg uchun qo'shasiz.

```go
import (
    "github.com/go-playground/locales/ru"
    ru_translations "github.com/go-playground/validator/v10/translations/ru"
)

uni := ut.New(en.New(), en.New(), ru.New())
transEN, _ := uni.GetTranslator("en")
transRU, _ := uni.GetTranslator("ru")
en_translations.RegisterDefaultTranslations(v, transEN)
ru_translations.RegisterDefaultTranslations(v, transRU)

// Validate(s any, lang string) — lang bo'yicha translator tanlanadi:
func (vd *Validator) Validate(s any, lang string) []FieldError {
    trans, found := vd.uni.GetTranslator(lang)
    if !found { trans = vd.fallback }
    ...
}
```

Tilni handler `Accept-Language`dan oladi — [17-i18n.md](17-i18n.md).

## Tez ishlatiladigan built-in teglar

| Teg | Ma'no |
|-----|-------|
| `required` | bo'sh bo'lmasin (0, "", nil) |
| `omitempty` | bo'sh bo'lsa qolgan qoidalarni tekshirma |
| `email`, `url`, `uuid`, `e164` | format (e164 — telefon) |
| `min=3,max=16` | string uzunligi / son qiymati |
| `len=7` | aniq uzunlik |
| `oneof=draft published` | enum |
| `gte=0,lte=100` | son chegarasi |
| `dive` | slice elementlarini tekshir: `validate:"required,dive,email"` |
| `required_if=Type custom` | shartli required |
| `eqfield=Password` | boshqa field'ga teng (confirm password) |
| `datetime=2006-01-02` | sana formati (string) |

## Handler'da ishlatish

```go
var req dto.CreateProductRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    response.Error(w, http.StatusBadRequest, "invalid json body")
    return
}
if ferrs := h.validator.Validate(req); ferrs != nil {
    response.ValidationError(w, ferrs) // 422
    return
}
p, err := h.svc.Create(r.Context(), req.ToInput(userID))
```

To'liq handler — [09-handler-router.md](09-handler-router.md).
