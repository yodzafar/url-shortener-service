package app

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yodzafar/url-shortener-service/internal/config"
	"github.com/yodzafar/url-shortener-service/internal/repository/postgres/repo"
	"github.com/yodzafar/url-shortener-service/internal/repository/postgres/sqlc"
	"github.com/yodzafar/url-shortener-service/internal/service"
	httptransport "github.com/yodzafar/url-shortener-service/internal/transport/http"
	"github.com/yodzafar/url-shortener-service/internal/transport/http/handler"
	"github.com/yodzafar/url-shortener-service/pkg/hash"
	"github.com/yodzafar/url-shortener-service/pkg/logger"
	"github.com/yodzafar/url-shortener-service/pkg/postgres"
	"github.com/yodzafar/url-shortener-service/pkg/validator"
	"golang.org/x/crypto/bcrypt"
)

var infraSet = wire.NewSet(
	providerLogger,
	providePool,
	provideHasher,
	validator.New,
	wire.Bind(new(sqlc.DBTX), new(*pgxpool.Pool)),
	sqlc.New,
	wire.Bind(new(service.PasswordHasher), new(*hash.Bcrypt)),
)

var repositorySet = wire.NewSet(
	repo.NewUserRepository,
	wire.Bind(new(service.UserRepository), new(*repo.UserRepository)),
)

var serviceSet = wire.NewSet(
	service.NewUserService,
)

var transportSet = wire.NewSet(
	handler.NewUserHandler,
	wire.Struct(new(httptransport.RouterDeps), "*"),
	httptransport.NewRouter,
	provideServer,
)

func providerLogger(cfg *config.Config) *slog.Logger {
	return logger.New(cfg.App.LogLevel, cfg.IsLocal())
}

func providePool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, func(), error) {
	pool, err := postgres.NewPool(ctx, cfg.DB.URL, cfg.DB.MaxConns)
	if err != nil {
		return nil, nil, err
	}

	return pool, func() { pool.Close() }, nil
}

func provideHasher() *hash.Bcrypt {
	return hash.NewBcrypt(bcrypt.DefaultCost)
}

func provideServer(cfg *config.Config, h http.Handler) *http.Server {
	return httptransport.NewServer(cfg.HTTP.Port, h, cfg.HTTP.ReadTimeout, cfg.HTTP.WriteTimeout)
}
