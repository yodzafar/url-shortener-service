package http

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/yodzafar/url-shortener-service/internal/transport/http/handler"
	"log/slog"
	"net/http"
)

type RouterDeps struct {
	user   *handler.UserHandler
	Logger *slog.Logger
}

func NewRouter(d RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)

}
