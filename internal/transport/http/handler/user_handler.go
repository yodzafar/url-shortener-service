package handler

import (
	"net/http"

	"github.com/yodzafar/url-shortener-service/internal/service"
	"github.com/yodzafar/url-shortener-service/internal/transport/http/dto"
	"github.com/yodzafar/url-shortener-service/internal/transport/http/response"
	"github.com/yodzafar/url-shortener-service/pkg/validator"
)

type UserHandler struct {
	svc       *service.UserService
	validator *validator.Validator
}

func NewUserHandler(svc *service.UserService, v *validator.Validator) *UserHandler {
	return &UserHandler{svc: svc, validator: v}
}

// Create GoDoc
// @Summary      Create User
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        request body     dto.CreateUserDto true "User information"
// @Success      201     {object} dto.UserResponse
// @Failure      400     {object} response.ErrorResponse
// @Failure      401     {object} response.ErrorResponse
// @Failure      422     {object} response.ErrorResponse
// @Security     BearerAuth
// @Router       /users [post]
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUserDto
	if err := decodeJson(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json body")
	}

	if ferrs := h.validator.Validate(req); ferrs != nil {
		response.ValidationError(w, ferrs)
		return
	}

	u, err := h.svc.Create(r.Context(), req.ToInput())

	if err != nil {
		response.FromError(w, r, err)
	}

	response.JSON(w, http.StatusOK, dto.FromUser(u))
}
