package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yodzafar/url-shortener-service/internal/domain"
)

type UserService struct {
	repo   UserRepository
	hasher PasswordHasher
}

func NewUserService(repo UserRepository, hasher PasswordHasher) *UserService {
	return &UserService{
		repo:   repo,
		hasher: hasher,
	}
}

type CreateUserInput struct {
	Email    string
	Password string
}

func (s *UserService) Create(ctx context.Context, input CreateUserInput) (*domain.User, error) {
	hash, err := s.hasher.Hash(input.Password)

	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()

	u := &domain.User{
		Email:        normalizeEmail(input.Email),
		PasswordHash: hash,
		Role:         domain.RoleUser,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	return u, nil
}

func normalizeEmail(e string) string {
	return strings.ToLower(strings.TrimSpace(e))
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}
