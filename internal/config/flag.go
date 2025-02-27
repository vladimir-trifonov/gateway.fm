package config

import (
	"github.com/spf13/cobra"
)

func InitFetcherFlags(cmd *cobra.Command) {
	cmd.Flags().String("fetcher.contract-address", "0xA13Ddb14437A8F34897131367ad3ca78416d6bCa", "set contract address")
	cmd.Flags().String("fetcher.event-topic", "0x3e54d0825ed78523037d00a81759237eb436ce774bd546993ee67a1b67b6e766", "set contract event topic to filter")
	cmd.Flags().Uint64("fetcher.from-block", 5158901, "the start block number for fetching events")
	cmd.Flags().Uint64("fetcher.to-block", 0, "the end block number for fetching events")
	cmd.Flags().Uint64("fetcher.chunk-size", 100, "the size of the block range chunks")
}

func InitBlockchainFlags(cmd *cobra.Command) {
	cmd.Flags().StringSlice("blockchain.rpc-urls", []string{"https://sepolia.infura.io"}, "set rpc urls")
	cmd.Flags().Uint64("blockchain.timeout", 6, "set timeout for rpc requests in seconds")
	cmd.Flags().Uint64("blockchain.health-check-freq", 30, "set frequency of RPC endpoint health checks in seconds")
}

func InitDatabaseFlags(cmd *cobra.Command) {
	cmd.Flags().String("database.path", "blockchain.db", "set the database path")
}

func InitLogFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("log.level", "l", "info", "logging level")
}
