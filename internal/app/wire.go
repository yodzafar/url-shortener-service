//go:build wireinject
// +build wireinject

package app

import (
	"context"

	"github.com/google/wire"
	"github.com/yodzafar/url-shortener-service/internal/config"
)

func InitApp(ctx context.Context, cfg *config.Config) (*App, func(), error) {
	wire.Build(infraSet, repositorySet, serviceSet, transportSet, NewApp)
	return nil, nil, nil
}
