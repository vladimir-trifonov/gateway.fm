package fetcher

import (
	"context"
	"sync"
	"time"

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
	defer wg.Wait()

	for chunk := range chunkChan {
		wg.Add(1)
		go func(c blockchain.BlockRange) {
			defer wg.Done()

			chunkCtx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.ChunkSize/10+30)*time.Second)
			defer cancel()

			logs, err := s.blockchain.FilterLogs(chunkCtx, contractAddress, s.cfg.EventTopic, c)

			if err != nil {
				log.Error("error processing chunk", "from", c.From, "to", c.To, "error", err)
				return
			}

			blockHashes := make([]common.Hash, 0)
			blockHashMap := make(map[common.Hash][]types.Log)

			for _, vLog := range logs {
				if vLog.Removed {
					continue
				}

				if _, exists := blockHashMap[vLog.BlockHash]; !exists {
					blockHashes = append(blockHashes, vLog.BlockHash)
				}
				blockHashMap[vLog.BlockHash] = append(blockHashMap[vLog.BlockHash], vLog)
			}

			headers, err := s.blockchain.BatchGetBlockHeaders(chunkCtx, blockHashes)
			if err != nil {
				log.Error("error fetching block headers", "error", err)
			}

			for blockHash, blockLogs := range blockHashMap {
				header := headers[blockHash]
				if header == nil {
					var fetchErr error
					header, fetchErr = s.blockchain.GetBlockHeader(chunkCtx, blockHash)
					if fetchErr != nil {
						log.Error("error fetching block header individually", "blockHash", blockHash.String(), "error", fetchErr)
						continue
					}
				}

				for _, vLog := range blockLogs {
					eventData := events.EventData{
						L1InfoRoot: vLog.Data,
						BlockTime:  header.Time,
						ParentHash: header.ParentHash,
					}

					log.Debug("event data", "eventData", eventData)

					if err := s.events.StoreEvent(eventData); err != nil {
						log.Error("event storing failed", "blockHash", blockHash.String(), "error", err)
					}
				}
			}
		}(chunk)
	}

	return nil
}
