package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/yodzafar/url-shortener-service/pkg/validator"
)

type ErrorBody struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Fields  []validator.FieldError `json:"fields,omitempty"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if v == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write response", "err", err)
	}
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, ErrorResponse{Error: ErrorBody{Code: codeFromStatus(status), Message: msg}})
}

func ValidationError(w http.ResponseWriter, fields []validator.FieldError) {
	JSON(w, http.StatusUnprocessableEntity, ErrorResponse{Error: ErrorBody{
		Code: "VALIDATION_ERROR", Message: "validation failed", Fields: fields,
	}})
}

func codeFromStatus(s int) string {
	switch s {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	default:
		return "INTERNAL_ERROR"
	}
}
