package service

import (
	"context"

	"github.com/yodzafar/url-shortener-service/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenManager interface {
	GenerateAccess(userID int64, role domain.Role) (string, error)
	GenerateRefresh() (string, error)
	ParseAccess(token string) (*domain.TokenClaims, error)
}
