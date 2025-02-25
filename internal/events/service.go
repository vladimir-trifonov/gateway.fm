package events

import (
	"encoding/binary"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
	bolt "go.etcd.io/bbolt"

	"gateway.fm/internal/database"
)

const BucketName = "SequenceBatches"

type Service struct {
	db         *database.Service
	bucketName string
	nextIndex  atomic.Uint64
}

func NewService(db *database.Service) *Service {
	nextIndexValue := db.InitBucket(BucketName)

	log.Debug("next index", "val", nextIndexValue)

	s := &Service{
		db:         db,
		bucketName: BucketName,
	}
	s.nextIndex.Store(nextIndexValue)

	return s
}

func (s *Service) Close() error {
	return s.db.Close()
}

func (s *Service) StoreEvent(event EventData) error {
	value, err := event.MarshalBinary()
	if err != nil {
		log.Error("error marshaling event data", "error", err)
		return err
	}

	key := s.nextIndex.Add(1) - 1

	log.Debug("storing event with key", "key", key)

	if err := s.storeEvent(key, value); err != nil {
		return err
	}

	return nil
}

func (s *Service) storeEvent(key uint64, value []byte) error {
	keyBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(keyBytes, key)

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.bucketName))
		return b.Put(keyBytes, value)
	})
}
