package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yodzafar/url-shortener-service/internal/domain"
	"github.com/yodzafar/url-shortener-service/internal/service"
)

var _ service.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	db DB
}

func NewUserRepository(db DB) *UserRepository {
	return &UserRepository{db: db}
}

const userColumns = `id, email, password_hash, role, created_at, updated_at`

// Create implements [service.UserRepository].
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	const q = `
	INSERT INTO users (email, password_hash, role, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5)
	RETURING id
	`

	err := r.db.QueryRow(ctx, q, u.Email, u.PasswordHash, u.Role, u.CreatedAt, u.UpdatedAt).Scan(&u.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrEmailTaken
		}
		return fmt.Errorf("user repo: create: %w", err)
	}

	return nil
}

// GetByEmail implements [service.UserRepository].
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.getOne(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email)
}

// GetByID implements [service.UserRepository].
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	return r.getOne(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
}

func (r *UserRepository) getOne(ctx context.Context, q string, args ...any) (*domain.User, error) {
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("user repo: query: %w", err)
	}
	u, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.User])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("user repo: scan: %w", err)
	}

	return &u, nil
}
