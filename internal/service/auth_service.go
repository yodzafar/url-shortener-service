package service

import (
	"context"

	"github.com/yodzafar/url-shortener-service/internal/domain"
)

type AuthService struct {
	userRepo UserRepository
	hasher   PasswordHasher
}

func NewAuthService(userRepo UserRepository, hasher PasswordHasher) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		hasher:   hasher,
	}
}

func (svc *AuthService) GenerateAccess(ctx context.Context, userID int64, role domain.Role) (string, error) {

	return "", nil
}
