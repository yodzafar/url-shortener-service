package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yodzafar/url-shortener-service/internal/domain"
	"github.com/yodzafar/url-shortener-service/internal/service"
)

var _ service.TokenManager = (*JWTManager)(nil)

type JWTManager struct {
	secret     []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	issuer     string
}

func NewManager(secret []byte, accessTTL, refreshTTL time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		secret:     secret,
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
		issuer:     issuer,
	}
}

type jwtClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func (m *JWTManager) GenerateAccess(userID int64, role domain.Role) (string, error) {
	now := time.Now()
	c := jwtClaims{
		Role:      string(role),
		Subject:   fmt.Sprint(userID),
		Issuer:    m.issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.AccessTTL)),
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

func (m *JWTManager) ParseAccess(tokenStr string) (*domain.TokenClaims, error) {
	var c jwtClaims
	tok, err := jwt.ParseWithClaims(tokenStr, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(m.issuer), jwt.WithExpirationRequired())
	if err != nil || !tok.Valid {
		return nil, domain.ErrInvalidToken
	}

	var userID int64
	if _, err := fmt.Sscan(c.Subject, &userID); err != nil {
		return nil, domain.ErrInvalidToken
	}

	return &domain.TokenClaims{UserID: userID, Role: domain.Role(c.Role)}, nil
}

func (m *JWTManager) GenerateRefresh() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b), nil
}
