package fetcher

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"gateway.fm/internal/blockchain"
	"gateway.fm/internal/config"
	"gateway.fm/internal/events"
	"gateway.fm/internal/utils"
)

type (
	Service struct {
		cfg        config.Fetcher
		events     *events.Service
		blockchain *blockchain.Service
	}
)

func NewService(cfg config.Fetcher, events *events.Service, blockchain *blockchain.Service) (*Service, error) {
	return &Service{
		cfg:        cfg,
		events:     events,
		blockchain: blockchain,
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	contractAddress := common.HexToAddress(s.cfg.ContractAddress)

	if err := s.blockchain.CheckIsContract(ctx, contractAddress); err != nil {
		log.Crit("not a contract or failed to check is a contract", "contractAddress", contractAddress.Hex())
	}

	latestBlock := s.cfg.ToBlock
	if latestBlock == 0 {
		latestBlock = s.blockchain.GetLatestBlockNumber()
	}

	if latestBlock < s.cfg.FromBlock {
		log.Crit("latest block number is less than the start block number", "latestBlock", latestBlock, "fromBlock", s.cfg.FromBlock)
	}

	chunks := utils.GetBlockRangeChunks(s.cfg.FromBlock, s.cfg.ToBlock, s.cfg.ChunkSize)

	chunkChan := make(chan blockchain.BlockRange, len(chunks))
	for _, chunk := range chunks {
		chunkChan <- chunk
	}
	close(chunkChan)

	var wg sync.WaitGroup
	resultsChan := make(chan []types.Log, len(chunks))
	errChan := make(chan error, len(chunks))

	defer func() {
		wg.Wait()
		close(resultsChan)
		close(errChan)
	}()

	for chunk := range chunkChan {
		wg.Add(1)
		go func(c blockchain.BlockRange) {
			defer wg.Done()

			logs, err := s.blockchain.FilterLogs(ctx, contractAddress, s.cfg.EventTopic, c)

			if err != nil {
				errChan <- fmt.Errorf("chunk %v failed: %w", c, err)
				return
			}

			resultsChan <- logs
		}(chunk)
	}

	for {
		select {
		case err, ok := <-errChan:
			if !ok {
				break
			}

			log.Error("error processing chunk", "error", err)
		case logs, ok := <-resultsChan:
			if !ok {
				return nil
			}

			for _, vLog := range logs {
				if vLog.Removed {
					continue
				}

				if err := s.processEvent(ctx, vLog); err != nil {
					log.Error("error processing event", "error", err)
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *Service) processEvent(ctx context.Context, vLog types.Log) error {
	header, err := s.blockchain.GetBlockHeader(ctx, vLog.BlockHash)
	if err != nil {
		return fmt.Errorf("failed to get block header: %w", err)
	}

	eventData := events.EventData{
		L1InfoRoot: vLog.Data,
		BlockTime:  header.Time,
		ParentHash: header.ParentHash,
	}

	log.Debug("processing event", "eventData", eventData)

	if err := s.events.StoreEvent(eventData); err != nil {
		return fmt.Errorf("event storing failed: %w", err)
	}

	return nil
}
