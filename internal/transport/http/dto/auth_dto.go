package dto

import "github.com/yodzafar/url-shortener-service/internal/domain"

// LoginDTO ---------- Request ----------
type LoginDTO struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
} // @name LoginDTO

// LoginResponse ---------- Response ----------
type LoginResponse struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
} // @name LoginResponse

func FromTokenPair(t *domain.TokenPair) LoginResponse {
	return LoginResponse{
		Access:  t.AccessToken,
		Refresh: t.RefreshToken,
	}
}
