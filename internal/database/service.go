package database

import (
	"encoding/binary"
	"log"

	bolt "go.etcd.io/bbolt"

	"gateway.fm/internal/config"
)

type (
	Service struct {
		db *bolt.DB
	}
)

func OpenConnection(cfg config.Database) *Service {
	db, err := bolt.Open(cfg.Path, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	return &Service{
		db: db,
	}
}

func (s *Service) Close() error {
	return s.db.Close()
}

func (s *Service) InitBucket(bucketName string) uint64 {
	var nextIndex uint64

	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}

		cursor := b.Cursor()
		lastKey, _ := cursor.Last()

		if lastKey != nil {
			nextIndex = binary.BigEndian.Uint64(lastKey) + 1
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	return nextIndex
}

func (s *Service) Update(fn func(tx *bolt.Tx) error) error {
	return s.db.Update(fn)
}
