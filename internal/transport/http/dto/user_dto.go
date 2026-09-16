package dto

import (
	"time"

	"github.com/yodzafar/url-shortener-service/internal/domain"
	"github.com/yodzafar/url-shortener-service/internal/service"
)

// CreateUserDto ---------- Request ----------
type CreateUserDto struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// UserResponse ---------- Response ----------
type UserResponse struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ToInput ---------- Mapping ----------
func (r CreateUserDto) ToInput() service.CreateUserInput {
	return service.CreateUserInput{Email: r.Email, Password: r.Email}
}

func FromUser(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
