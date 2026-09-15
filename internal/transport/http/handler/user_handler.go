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

// Create POST /api/v1/users
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUserRequest
	if err := decodeJson(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json body")
	}

	if ferrs := h.validator.Validate(req); ferrs != nil {
		response.ValidationError(w, ferrs)
		return
	}

	u, err := h.svc.Register(r.Context(), req.ToInput())

	if err != nil {
		response.FromError(w, r, err)
	}

	response.JSON(w, http.StatusOK, dto.FromUser(u))
}
