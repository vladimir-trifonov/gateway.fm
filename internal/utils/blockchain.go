package utils

import "gateway.fm/internal/blockchain"

func GetBlockRangeChunks(start, end, chunkSize uint64) []blockchain.BlockRange {
	var chunks []blockchain.BlockRange
	for from := start; from <= end; from += chunkSize {
		to := from + chunkSize - 1
		if to > end {
			to = end
		}
		chunks = append(chunks, blockchain.BlockRange{From: from, To: to})
	}
	return chunks
}
