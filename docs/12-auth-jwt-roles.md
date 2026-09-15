# 12 — Auth: JWT, Refresh token, Rollar (RBAC)

## Oqim

```
POST /auth/register  → email+password → bcrypt hash → users jadvali
POST /auth/login     → parolni tekshir → access JWT (15m) + refresh token (30d)
GET  /products (Authorization: Bearer <access>) → middleware JWT'ni tekshiradi → user context'ga
POST /auth/refresh   → refresh token → yangi access + refresh (rotation)
POST /auth/logout    → refresh tokenni o'chir
```

- **Access token** — JWT, qisqa muddat, stateless (DB'ga bormaydi). Ichida `user_id`, `role`.
- **Refresh token** — tasodifiy string, uzoq muddat, **DB/Redis'da saqlanadi** (bekor qilish mumkin).

Kutubxonalar: `github.com/golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`.

## `pkg/hash/bcrypt.go`

```go
package hash

import "golang.org/x/crypto/bcrypt"

type Bcrypt struct{ cost int }

func NewBcrypt() *Bcrypt { return &Bcrypt{cost: bcrypt.DefaultCost} } // 10; prod'da 12

func (b *Bcrypt) Hash(password string) (string, error) {
    h, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
    return string(h), err
}

func (b *Bcrypt) Compare(hash, password string) error {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
```

## `pkg/jwt/jwt.go` — TokenManager

```go
package jwt

import (
    "crypto/rand"
    "encoding/base64"
    "errors"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"

    "github.com/yodzafar/myservice/internal/domain"
    "github.com/yodzafar/myservice/internal/service"
)

type Manager struct {
    secret    []byte
    accessTTL time.Duration
    issuer    string
}

func NewManager(secret string, accessTTL time.Duration, issuer string) *Manager {
    return &Manager{secret: []byte(secret), accessTTL: accessTTL, issuer: issuer}
}

type claims struct {
    Role string `json:"role"`
    jwt.RegisteredClaims
}

func (m *Manager) GenerateAccess(userID int64, role domain.Role) (string, error) {
    now := time.Now()
    c := claims{
        Role: string(role),
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   fmt.Sprint(userID),
            Issuer:    m.issuer,
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
        },
    }
    return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

func (m *Manager) ParseAccess(tokenStr string) (*service.TokenClaims, error) {
    var c claims
    tok, err := jwt.ParseWithClaims(tokenStr, &c, func(t *jwt.Token) (any, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok { // alg confusion hujumidan himoya
            return nil, errors.New("unexpected signing method")
        }
        return m.secret, nil
    }, jwt.WithValidMethods([]string{"HS256"}), // jwt v5: kutubxona "strongly encouraged" deydi — alg'ni parser tekshiradi
        jwt.WithIssuer(m.issuer), jwt.WithExpirationRequired())
    if err != nil || !tok.Valid {
        return nil, domain.ErrInvalidToken
    }
    var userID int64
    if _, err := fmt.Sscan(c.Subject, &userID); err != nil {
        return nil, domain.ErrInvalidToken
    }
    return &service.TokenClaims{UserID: userID, Role: domain.Role(c.Role)}, nil
}

// Refresh — tasodifiy 32 bayt, JWT emas
func (m *Manager) GenerateRefresh() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(b), nil
}
```

`pkg/jwt` `internal/domain`ni import qilyapti — demak aslida `internal/auth`ga tegishli. O'rganish uchun ok; toza qilmoqchi bo'lsangiz `Role`ni `string` sifatida uzating.

## Refresh token saqlash — `service/ports.go`

```go
type RefreshTokenStore interface {
    Save(ctx context.Context, token string, userID int64, ttl time.Duration) error
    Get(ctx context.Context, token string) (userID int64, err error) // topilmasa domain.ErrInvalidToken
    Delete(ctx context.Context, token string) error
}
```

Implementatsiya: Redis (`refresh:<token>` → userID, TTL bilan) — [18-cache-redis.md](18-cache-redis.md), yoki Postgres `refresh_tokens` jadvali.

## `internal/service/auth_service.go`

```go
package service

import (
    "context"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/yodzafar/myservice/internal/domain"
)

type AuthService struct {
    users      UserRepository
    hasher     PasswordHasher
    tokens     TokenManager
    refresh    RefreshTokenStore
    refreshTTL time.Duration
    clock      Clock
}

func NewAuthService(users UserRepository, hasher PasswordHasher, tokens TokenManager,
    refresh RefreshTokenStore, refreshTTL time.Duration, clock Clock) *AuthService {
    return &AuthService{users: users, hasher: hasher, tokens: tokens,
        refresh: refresh, refreshTTL: refreshTTL, clock: clock}
}

type TokenPair struct {
    AccessToken  string
    RefreshToken string
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.User, error) {
    hash, err := s.hasher.Hash(password)
    if err != nil {
        return nil, fmt.Errorf("hash password: %w", err)
    }
    now := s.clock.Now()
    u := &domain.User{
        Email: strings.ToLower(strings.TrimSpace(email)), PasswordHash: hash,
        Role: domain.RoleUser, CreatedAt: now, UpdatedAt: now,
    }
    if err := s.users.Create(ctx, u); err != nil {
        return nil, err // ErrEmailTaken → 409
    }
    return u, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
    u, err := s.users.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
    if errors.Is(err, domain.ErrUserNotFound) {
        return nil, domain.ErrInvalidCredentials // "email yo'q" demaymiz — enumeration'dan himoya
    }
    if err != nil {
        return nil, err
    }
    if err := s.hasher.Compare(u.PasswordHash, password); err != nil {
        return nil, domain.ErrInvalidCredentials
    }
    return s.issueTokens(ctx, u)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
    userID, err := s.refresh.Get(ctx, refreshToken)
    if err != nil {
        return nil, domain.ErrInvalidToken
    }
    _ = s.refresh.Delete(ctx, refreshToken) // rotation: eski token bir marta ishlaydi

    u, err := s.users.GetByID(ctx, userID)
    if err != nil {
        return nil, domain.ErrInvalidToken
    }
    return s.issueTokens(ctx, u)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
    return s.refresh.Delete(ctx, refreshToken)
}

// Middleware har request'da chaqiradi: claims → user
func (s *AuthService) UserFromToken(ctx context.Context, accessToken string) (*domain.User, error) {
    c, err := s.tokens.ParseAccess(accessToken)
    if err != nil {
        return nil, err
    }
    // Variant A (tez): DB'ga bormasdan claims'dan user quramiz
    return &domain.User{ID: c.UserID, Role: c.Role}, nil
    // Variant B (aniq, sekinroq): return s.users.GetByID(ctx, c.UserID) — bloklangan userni darhol to'xtatadi
}

func (s *AuthService) issueTokens(ctx context.Context, u *domain.User) (*TokenPair, error) {
    access, err := s.tokens.GenerateAccess(u.ID, u.Role)
    if err != nil {
        return nil, fmt.Errorf("generate access: %w", err)
    }
    refresh, err := s.tokens.GenerateRefresh()
    if err != nil {
        return nil, fmt.Errorf("generate refresh: %w", err)
    }
    if err := s.refresh.Save(ctx, refresh, u.ID, s.refreshTTL); err != nil {
        return nil, fmt.Errorf("save refresh: %w", err)
    }
    return &TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}
```

## Auth middleware — `internal/transport/http/middleware/auth.go`

```go
package middleware

import (
    "context"
    "net/http"
    "strings"

    "github.com/yodzafar/myservice/internal/domain"
    "github.com/yodzafar/myservice/internal/service"
    "github.com/yodzafar/myservice/internal/transport/http/response"
)

type Auth struct {
    auth *service.AuthService
}

func NewAuth(auth *service.AuthService) *Auth { return &Auth{auth: auth} }

// Authenticate — "kim?" Token yo'q/noto'g'ri → 401
func (a *Auth) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        h := r.Header.Get("Authorization")
        if !strings.HasPrefix(h, "Bearer ") {
            response.Error(w, http.StatusUnauthorized, "missing bearer token")
            return
        }
        user, err := a.auth.UserFromToken(r.Context(), strings.TrimPrefix(h, "Bearer "))
        if err != nil {
            response.Error(w, http.StatusUnauthorized, "invalid or expired token")
            return
        }
        next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
    })
}

// RequireRole — "mumkinmi?" Rol mos kelmasa → 403. Authenticate'dan KEYIN.
func (a *Auth) RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
    allowed := make(map[domain.Role]struct{}, len(roles))
    for _, r := range roles {
        allowed[r] = struct{}{}
    }
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            u := UserFromContext(r.Context())
            if u == nil {
                response.Error(w, http.StatusUnauthorized, "unauthorized")
                return
            }
            if _, ok := allowed[u.Role]; !ok {
                response.Error(w, http.StatusForbidden, "insufficient role")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// ---- context helpers ----
type ctxKey int

const userKey ctxKey = iota

// WithUser — exported: handler testlarida ham ishlatiladi
func WithUser(ctx context.Context, u *domain.User) context.Context {
    return context.WithValue(ctx, userKey, u)
}

func UserFromContext(ctx context.Context) *domain.User {
    u, _ := ctx.Value(userKey).(*domain.User)
    return u
}
```

Router'da: [09-handler-router.md](09-handler-router.md) — `r.Use(authMW.Authenticate)` / `r.Use(authMW.Authenticate, authMW.RequireRole(domain.RoleAdmin))`.

## Auth DTO + handler (qisqa)

```go
// dto/auth_dto.go
type RegisterRequest struct {
    Email    string `json:"email"    validate:"required,email,max=255"`
    Password string `json:"password" validate:"required,min=8,max=72,strong_password"` // bcrypt max 72 bayt
}
type LoginRequest struct {
    Email    string `json:"email"    validate:"required,email"`
    Password string `json:"password" validate:"required"`
}
type RefreshRequest struct {
    RefreshToken string `json:"refresh_token" validate:"required"`
}
type TokenResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    TokenType    string `json:"token_type"` // "Bearer"
}
type UserResponse struct {
    ID    int64  `json:"id"`
    Email string `json:"email"`
    Role  string `json:"role"`
}

// handler/auth_handler.go
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req dto.LoginRequest
    if err := decodeJSON(r, &req); err != nil {
        response.Error(w, http.StatusBadRequest, "invalid json body")
        return
    }
    if ferrs := h.validator.Validate(req); ferrs != nil {
        response.ValidationError(w, ferrs)
        return
    }
    pair, err := h.svc.Login(r.Context(), req.Email, req.Password)
    if err != nil {
        response.FromError(w, r, err)
        return
    }
    response.JSON(w, http.StatusOK, dto.TokenResponse{
        AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, TokenType: "Bearer",
    })
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
    u := middleware.UserFromContext(r.Context())
    response.JSON(w, http.StatusOK, dto.UserResponse{ID: u.ID, Email: u.Email, Role: string(u.Role)})
}
```

## Ruxsat modellari

| Model | Qachon | Qayerda tekshiriladi |
|-------|--------|---------------------|
| **RBAC** (rol) | admin/user/manager — sodda | middleware `RequireRole` |
| **Ownership** | o'z resursini o'zgartirish | service: `p.CanBeEditedBy(actor)` |
| **Permission** (granular) | `product:write`, `user:ban` | JWT'da `permissions: []`, middleware `RequirePermission("product:write")` |
| **ABAC / Casbin** | murakkab siyosat | `github.com/casbin/casbin/v2` |

Boshlang'ich: RBAC (middleware) + ownership (service). Ko'p rol/permission bo'lsa — permission'lar jadvali va Casbin.

## Xavfsizlik checklist

- Secret ≥ 32 belgi, env'dan. Kodga yozmang.
- `alg` tekshiruvi (`jwt.WithValidMethods` + `SigningMethodHMAC`) — `none`/`RS256` almashtirish hujumi.
- Access TTL qisqa (5–15m). Refresh rotation (eski token bir marta).
- Parol: bcrypt/argon2, hech qachon SHA256. `password_hash` DTO'da chiqmasin.
- Login xatosi bir xil: "invalid email or password" (email bormi bilib bo'lmasin).
- Login'ga rate limit (`httprate`, 5 req/min per IP).
- Refresh token: HttpOnly cookie (web) yoki body (mobile). Web'da localStorage'ga qo'ymang.
- Prod'da faqat HTTPS.
- Muhim amallarda (parol o'zgartirish) barcha refresh tokenlarni o'chiring.
