package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"gateway.fm/internal/config"
)

type BlockRange struct {
	From uint64
	To   uint64
}

type (
	Service struct {
		cfg               config.Blockchain
		clients           []*ethclient.Client
		latestBlockNumber uint64
		headerCache       sync.Map
	}
)

func NewClient(cfg config.Blockchain) (*Service, error) {
	clients := make([]*ethclient.Client, 0)
	latestBlockNumber := uint64(0)

	for _, url := range cfg.RpcUrls {
		client, err := ethclient.Dial(url)
		if err != nil {
			return nil, err
		}

		blockNumber, err := client.BlockNumber(context.Background())
		if err != nil {
			log.Warn("failed to get block number from url", "url", url, "error", err)
			continue
		}

		clients = append(clients, client)
		if blockNumber > latestBlockNumber {
			latestBlockNumber = blockNumber
		}
	}

	return &Service{
		cfg:               cfg,
		clients:           clients,
		latestBlockNumber: latestBlockNumber,
	}, nil
}

func (s *Service) GetLatestBlockNumber() uint64 {
	return s.latestBlockNumber
}

func (s *Service) CheckIsContract(ctx context.Context, contractAddress common.Address) error {
	if len(s.clients) == 0 {
		return fmt.Errorf("no clients available")
	}

	code, err := s.clients[0].CodeAt(context.Background(), contractAddress, nil)
	if err != nil {
		return err
	}

	if len(code) == 0 {
		return fmt.Errorf("no code at address %s", &contractAddress)
	}

	return nil
}

func (s *Service) FilterLogs(ctx context.Context, contractAddress common.Address, eventTopic string, chunk BlockRange) ([]types.Log, error) {
	type result struct {
		logs []types.Log
		err  error
	}

	log.Debug("filtering logs ", "contractAddress", contractAddress.String(), "from", chunk.From, "to", chunk.To)

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(chunk.From)),
		ToBlock:   big.NewInt(int64(chunk.To)),
		Addresses: []common.Address{
			contractAddress,
		},
		Topics: [][]common.Hash{
			{
				common.HexToHash(eventTopic),
			},
		},
	}

	resultCh := make(chan result, len(s.clients))

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.Timeout)*time.Second)
	defer cancel()

	for _, client := range s.clients {
		go func(c *ethclient.Client) {
			logs, err := c.FilterLogs(ctx, query)
			resultCh <- result{logs, err}
		}(client)
	}

	var firstErr error
	for i := 0; i < len(s.clients); i++ {
		res := <-resultCh
		if res.err == nil && res.logs != nil {
			return res.logs, nil
		} else {
			firstErr = res.err
		}
	}

	return nil, fmt.Errorf("all endpoints failed on filtering longs, last error: %v", firstErr)
}

func (s *Service) GetBlockHeader(ctx context.Context, hash common.Hash) (*types.Header, error) {
	if val, ok := s.headerCache.Load(hash); ok {
		return val.(*types.Header), nil
	}

	type result struct {
		header *types.Header
		err    error
	}

	resultCh := make(chan result, len(s.clients))

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.Timeout)*time.Second)
	defer cancel()

	for _, client := range s.clients {
		go func(c *ethclient.Client) {
			header, err := c.HeaderByHash(ctx, hash)
			resultCh <- result{header, err}
		}(client)
	}

	var firstErr error
	for i := 0; i < len(s.clients); i++ {
		res := <-resultCh
		if res.err == nil && res.header != nil {
			s.headerCache.Store(hash, res.header)
			return res.header, nil
		} else {
			firstErr = res.err
		}
	}

	return nil, fmt.Errorf("all endpoints failed of fetching block header, last error: %v", firstErr)
}
