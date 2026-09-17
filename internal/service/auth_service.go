package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/yodzafar/url-shortener-service/internal/domain"
)

type AuthService struct {
	userRepo UserRepository
	hasher   PasswordHasher
	token    TokenManager
}

func NewAuthService(userRepo UserRepository, hasher PasswordHasher, token TokenManager) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		hasher:   hasher,
		token:    token,
	}
}

func (svc *AuthService) Login(ctx context.Context, email, password string) (*domain.TokenPair, error) {
	u, err := svc.userRepo.GetByEmail(ctx, email)

	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if check := svc.hasher.Compare(u.PasswordHash, password); check != nil {
		return nil, domain.ErrInvalidCredentials
	}

	return svc.issueTokens(u)
}

func (svc *AuthService) UserFromToken(accessToken string) (*domain.User, error) {
	c, err := svc.token.ParseAccess(accessToken)

	if err != nil {
		return nil, err
	}

	return &domain.User{ID: c.UserID, Role: c.Role}, nil
}

func (svc *AuthService) issueTokens(user *domain.User) (*domain.TokenPair, error) {
	access, err := svc.token.GenerateAccess(user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate access token failed: %w", err)
	}

	refresh, err := svc.token.GenerateRefresh()

	if err != nil {
		return nil, fmt.Errorf("generate refresh token failed: %w", err)
	}

	return &domain.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}
