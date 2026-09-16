package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/yodzafar/url-shortener-service/internal/domain"
	"github.com/yodzafar/url-shortener-service/internal/repository/postgres/sqlc"
	"github.com/yodzafar/url-shortener-service/internal/service"
)

var _ service.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	q *sqlc.Queries
}

func NewUserRepository(q *sqlc.Queries) *UserRepository {
	return &UserRepository{q: q}
}

const userColumns = `id, email, password_hash, role, created_at, updated_at`

// Create implements [service.UserRepository].
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {

	id, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         string(u.Role),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	})

	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailTaken
		}
		return fmt.Errorf("user repo: create: %w", err)
	}

	u.ID = id

	return nil
}

// GetByEmail implements [service.UserRepository].
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("user repo: get by email: %w", err)
	}

	return ToDomainUser(row), err
}

// GetByID implements [service.UserRepository].
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	row, err := r.q.GetUserByID(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("user repo: get by id: %w", err)
	}

	return ToDomainUser(row), err
}
