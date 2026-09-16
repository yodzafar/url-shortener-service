package repo

import (
	"github.com/yodzafar/url-shortener-service/internal/domain"
	"github.com/yodzafar/url-shortener-service/internal/repository/postgres/sqlc"
)

func ToDomainUser(r sqlc.User) *domain.User {
	return &domain.User{
		ID:           r.ID,
		Email:        r.Email,
		PasswordHash: r.PasswordHash,
		Role:         domain.Role(r.Role),
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}
