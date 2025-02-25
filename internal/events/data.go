package events

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
)

type EventData struct {
	L1InfoRoot []byte
	BlockTime  uint64
	ParentHash common.Hash
}

func (e *EventData) MarshalBinary() ([]byte, error) {
	b := make([]byte, 72)
	copy(b[0:32], e.L1InfoRoot[:])
	binary.BigEndian.PutUint64(b[32:40], e.BlockTime)
	copy(b[40:72], e.ParentHash[:])
	return b, nil
}
