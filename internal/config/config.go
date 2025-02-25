package config

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type (
	Config struct {
		Fetcher    Fetcher
		Blockchain Blockchain
		Database   Database
		Log        Log
	}
)

type Fetcher struct {
	ContractAddress string `mapstructure:"contract-address" validate:"required"`
	EventTopic      string `mapstructure:"event-topic" validate:"required"`
	FromBlock       uint64 `mapstructure:"from-block" validate:"required"`
	ToBlock         uint64 `mapstructure:"to-block"`
	ChunkSize       uint64 `mapstructure:"chunk-size" validate:"required"`
}

type Blockchain struct {
	RpcUrls []string `mapstructure:"rpc-urls" validate:"dive,url,required"`
	Timeout uint64   `mapstructure:"timeout"`
}

type Database struct {
	Path string `mapstructure:"path" validate:"required"`
}

type Log struct {
	Level string `mapstructure:"level" validate:"oneof=debug info warn error"`
}

func Init(cfg any, cmd *cobra.Command) error {
	if err := readCmdVariables(cmd); err != nil {
		return fmt.Errorf("failed to read command line variables: %w", err)
	}

	if err := unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return nil
}

func unmarshal(cfg any) error {
	hooks := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToSliceHookFunc(","),
	))

	return viper.Unmarshal(cfg, hooks)
}

func readCmdVariables(cmd *cobra.Command) error {
	var err error
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if err = viper.BindPFlag(f.Name, f); err != nil {
			return
		}
	})

	return err
}
