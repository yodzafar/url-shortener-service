package handler

import (
	"net/http"

	"github.com/yodzafar/url-shortener-service/internal/service"
	"github.com/yodzafar/url-shortener-service/internal/transport/http/dto"
	"github.com/yodzafar/url-shortener-service/internal/transport/http/response"
	"github.com/yodzafar/url-shortener-service/pkg/validator"
)

type AuthHandler struct {
	authService *service.AuthService
	validator   *validator.Validator
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login GoDoc
// @Summary      Login
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body     dto.LoginDTO true "user credentials"
// @Success      200     {object} dto.LoginResponse
// @Failure      400     {object} response.ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginDTO

	if err := decodeJson(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if ferrs := h.validator.Validate(req); ferrs != nil {
		response.ValidationError(w, ferrs)
		return
	}

	tokens, err := h.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		response.FromError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.FromTokenPair(tokens))
}
