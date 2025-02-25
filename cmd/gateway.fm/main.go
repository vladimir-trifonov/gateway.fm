package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"gateway.fm/internal/config"
)

var Version = "N/A"
var cfg = new(config.Config)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	root := &cobra.Command{
		Use:     "gateway.fm",
		Version: Version,
		Short:   "gateway.fm",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.Init(cfg, cmd); err != nil {
				return fmt.Errorf("failed to init config: %w", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			InitLogger(ctx, cfg.Log.Level)

			fx.New(BootstrapApp()).Run()

			return nil
		},
	}

	config.InitLogFlags(root)
	config.InitDatabaseFlags(root)
	config.InitBlockchainFlags(root)
	config.InitFetcherFlags(root)

	cobra.CheckErr(root.ExecuteContext(ctx))
}
