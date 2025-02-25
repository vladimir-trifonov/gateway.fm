package main

import (
	"context"

	"go.uber.org/fx"

	"gateway.fm/internal/blockchain"
	"gateway.fm/internal/config"
	"gateway.fm/internal/database"
	"gateway.fm/internal/events"
	"gateway.fm/internal/fetcher"
)

func BootstrapApp() fx.Option {
	return fx.Module("app",
		fx.Provide(
			func() config.Fetcher { return cfg.Fetcher },
			func() config.Blockchain { return cfg.Blockchain },
			func() config.Database { return cfg.Database },
			database.OpenConnection,
			blockchain.NewClient,
			events.NewService,
			fetcher.NewService,
		),
		fx.Invoke(invokeFetcherService),
	)
}

func invokeFetcherService(
	lc fx.Lifecycle,
	service *fetcher.Service,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return service.Start(ctx)
		},
	})
}
