package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	_ "github.com/yodzafar/url-shortener-service/api/swagger"
	"github.com/yodzafar/url-shortener-service/internal/transport/http/handler"
	"github.com/yodzafar/url-shortener-service/internal/transport/http/middleware"
)

type RouterDeps struct {
	User   *handler.UserHandler
	Logger *slog.Logger
}

func NewRouter(d RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.ClientIPFromRemoteAddr)
	r.Use(middleware.Logger(d.Logger))
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorazation", "Content-type", "Accept-Language"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok"))

		if err != nil {

		}
	})
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/users", func(ur chi.Router) {
			ur.Post("/", d.User.Create)
			ur.Get("/{id}", d.User.GetById)
		})
	})

	return r
}
