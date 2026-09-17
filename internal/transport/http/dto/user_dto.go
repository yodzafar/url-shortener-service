package dto

import (
	"time"

	"github.com/yodzafar/url-shortener-service/internal/domain"
	"github.com/yodzafar/url-shortener-service/internal/service/input"
)

// CreateUserDto ---------- Request ----------
type CreateUserDto struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
} //@name CreateUserDto

// UserResponse ---------- Response ----------
type UserResponse struct {
	ID        int64     `json:"id" validate:"required"`
	Email     string    `json:"email" validate:"required"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
} //@name UserResponse

// ToInput ---------- Mapping ----------
func (r CreateUserDto) ToInput() input.CreateUserInput {
	return input.CreateUserInput{Email: r.Email, Password: r.Password}
}

func FromUser(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
