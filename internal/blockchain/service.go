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
		roundRobinMu      sync.Mutex
		roundRobinIndex   int
		clientHealth      []bool
		clientHealthMu    sync.RWMutex
	}
)

func NewClient(cfg config.Blockchain) (*Service, error) {
	clients := make([]*ethclient.Client, 0, len(cfg.RpcUrls))
	latestBlockNumber := uint64(0)
	clientHealth := make([]bool, len(cfg.RpcUrls))

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, url := range cfg.RpcUrls {
		wg.Add(1)
		go func(index int, endpoint string) {
			defer wg.Done()
			client, err := ethclient.Dial(endpoint)
			if err != nil {
				log.Warn("failed to connect to RPC", "url", endpoint, "error", err)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
			defer cancel()

			blockNumber, err := client.BlockNumber(ctx)
			if err != nil {
				log.Warn("failed to get block number from url", "url", endpoint, "error", err)
				return
			}

			mu.Lock()
			clients = append(clients, client)
			clientHealth[index] = true
			if blockNumber > latestBlockNumber {
				latestBlockNumber = blockNumber
			}
			mu.Unlock()
		}(i, url)
	}

	wg.Wait()

	if len(clients) == 0 {
		return nil, fmt.Errorf("no healthy RPC clients available")
	}

	s := &Service{
		cfg:               cfg,
		clients:           clients,
		latestBlockNumber: latestBlockNumber,
		clientHealth:      clientHealth,
	}

	go s.monitorClientHealth(cfg.Timeout)

	return s, nil
}

func (s *Service) monitorClientHealth(timeout uint64) {
	ticker := time.NewTicker(time.Duration(s.cfg.HealthCheckFreq) * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		for i, client := range s.clients {
			go func(index int, c *ethclient.Client) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
				defer cancel()

				_, err := c.BlockNumber(ctx)

				s.clientHealthMu.Lock()
				s.clientHealth[index] = err == nil
				s.clientHealthMu.Unlock()

				if err != nil {
					log.Warn("client health check failed", "index", index, "error", err)
				}
			}(i, client)
		}
	}
}

func (s *Service) getHealthyClient() (*ethclient.Client, error) {
	s.roundRobinMu.Lock()
	defer s.roundRobinMu.Unlock()

	s.clientHealthMu.RLock()
	defer s.clientHealthMu.RUnlock()

	startIndex := s.roundRobinIndex
	clientCount := len(s.clients)

	if clientCount == 0 {
		return nil, fmt.Errorf("no clients available")
	}

	for i, client := range s.clients {
		index := (startIndex + i) % clientCount
		if s.clientHealth[index] {
			s.roundRobinIndex = (index + 1) % clientCount
			return client, nil
		}
	}

	s.roundRobinIndex = (startIndex + 1) % clientCount
	return s.clients[startIndex], nil
}

func (s *Service) GetLatestBlockNumber() uint64 {
	return s.latestBlockNumber
}

func (s *Service) CheckIsContract(ctx context.Context, contractAddress common.Address) error {
	client, err := s.getHealthyClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.Timeout)*time.Second)
	defer cancel()

	code, err := client.CodeAt(ctx, contractAddress, nil)
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

	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.Timeout)*time.Second)
	defer cancel()

	resultCh := make(chan result)
	clientIndices := make([]int, 0)

	s.clientHealthMu.RLock()
	for i, healthy := range s.clientHealth {
		if healthy {
			clientIndices = append(clientIndices, i)
		}
	}
	s.clientHealthMu.RUnlock()

	for i := range clientIndices {
		go func(c *ethclient.Client) {
			logs, err := c.FilterLogs(ctx, query)
			resultCh <- result{logs, err}
		}(s.clients[i])
	}

	timer := time.NewTimer(time.Duration(s.cfg.Timeout) * time.Second)
	defer timer.Stop()

	var firstErr error
	responseCount := 0
	expectedResponses := len(clientIndices)

	for responseCount < expectedResponses {
		select {
		case res := <-resultCh:
			responseCount++
			if res.err == nil && res.logs != nil {
				return res.logs, nil
			} else if firstErr == nil {
				firstErr = res.err
			}
		case <-timer.C:
			if firstErr != nil {
				return nil, fmt.Errorf("timeout reached while filtering logs: %v", firstErr)
			}
			return nil, fmt.Errorf("timeout reached while filtering logs")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("all endpoints failed on filtering logs, last error: %v", firstErr)
}

func (s *Service) GetBlockHeader(ctx context.Context, hash common.Hash) (*types.Header, error) {
	if val, ok := s.headerCache.Load(hash); ok {
		return val.(*types.Header), nil
	}

	client, err := s.getHealthyClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.Timeout)*time.Second)
	defer cancel()

	header, err := client.HeaderByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get block header: %w", err)
	}

	if header != nil {
		s.headerCache.Store(hash, header)
	}

	return header, nil
}

func (s *Service) BatchGetBlockHeaders(ctx context.Context, hashes []common.Hash) (map[common.Hash]*types.Header, error) {
	result := make(map[common.Hash]*types.Header)
	var missingHashes []common.Hash

	for _, hash := range hashes {
		if val, ok := s.headerCache.Load(hash); ok {
			result[hash] = val.(*types.Header)
		} else {
			missingHashes = append(missingHashes, hash)
		}
	}

	if len(missingHashes) == 0 {
		return result, nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	client, err := s.getHealthyClient()
	if err != nil {
		return nil, err
	}

	for _, hash := range missingHashes {
		wg.Add(1)
		go func(h common.Hash) {
			defer wg.Done()

			fetchCtx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.Timeout)*time.Second)
			defer cancel()

			header, err := client.HeaderByHash(fetchCtx, h)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors = append(errors, err)
				return
			}

			result[h] = header
			s.headerCache.Store(h, header)
		}(hash)
	}

	wg.Wait()

	if len(errors) > 0 && len(errors) == len(missingHashes) {
		return result, fmt.Errorf("failed to fetch all missing headers: %v", errors[0])
	}

	return result, nil
}
