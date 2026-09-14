package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/yodzafar/url-shortener-service/internal/domain"
)

type UserService struct {
	repo  UserRepository
	clock Clock
	hasher PasswordHasher
	log   *slog.Logger
}

func NewUserRepository(repo UserRepository, clock Clock, log *slog.Logger) *UserService {
	return &UserService{
		repo:  repo,
		clock: clock,
		log:   log,
	}
}

type RegisterInput struct {
	Email string
	Password string
}

func(s*UserService) Register(ctx context.Context, input RegisterInput) (*domain.User, error) {
	hash, err: s.hasher.Hash(input.Password)

	if err != nil {
		return  nil, fmt.Errorf("hash password: %w", err)
	}
}
