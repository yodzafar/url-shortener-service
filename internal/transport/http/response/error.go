package response

import (
	"errors"
	"net/http"

	"github.com/yodzafar/url-shortener-service/internal/domain"
)

type mapping struct {
	status int
	code   string
}

var errMap = map[error]mapping{
	domain.ErrNotFound:           {http.StatusNotFound, "NOT_FOUND"},
	domain.ErrUserNotFound:       {http.StatusNotFound, "USER_NOT_FOUND"},
	domain.ErrEmailTaken:         {http.StatusConflict, "EMAIL_TAKEN"},
	domain.ErrInvalidCredentials: {http.StatusUnauthorized, "INVALID_CREDENTIALS"},
	domain.ErrInvalidToken:       {http.StatusUnauthorized, "INVALID_TOKEN"},
	domain.ErrForbidden:          {http.StatusForbidden, "FORBIDDEN"},
}

func FromError(w http.ResponseWriter, r *http.Request, err error) {
	for target, m := range errMap {
		if errors.Is(err, target) {
			JSON(w, m.status, ErrorResponse{Code: m.code, Message: target.Error()})
		}
	}
}
