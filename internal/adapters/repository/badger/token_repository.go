package badger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	badgerdb "github.com/dgraph-io/badger/v4"
)

const (
	tokenStorageVersion = "v1"
	tokenKind           = "Token"
	tokenPrefix         = "tokens:"
)

type TokenRepository struct {
	db *badgerdb.DB
}

func NewTokenRepository(db *badgerdb.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func tokenKey(hash string) []byte {
	return []byte(tokenPrefix + hash)
}

func (r *TokenRepository) marshal(token *domain.Token) ([]byte, error) {
	return json.Marshal(StoredResource[domain.Token]{
		StorageVersion: tokenStorageVersion,
		Kind:           tokenKind,
		Spec:           *token,
	})
}

func (r *TokenRepository) unmarshal(data []byte) (*domain.Token, error) {
	stored, err := unmarshal[domain.Token](data)
	if err != nil {
		return nil, err
	}
	return &stored.Spec, nil
}

func (r *TokenRepository) Find(ctx context.Context, hash string) (*domain.Token, error) {
	var token *domain.Token

	err := r.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(tokenKey(hash))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("token not found")
		}

		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			token, err = r.unmarshal(val)
			return err
		})
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}

func (r *TokenRepository) Save(ctx context.Context, token *domain.Token) error {
	data, err := r.marshal(token)
	if err != nil {
		return fmt.Errorf("marshall token: %w", err)
	}

	return r.db.Update(func(txn *badgerdb.Txn) error {
		entry := badgerdb.NewEntry(tokenKey(token.Hash), data)

		// badger evicts the record automatically once the token expires
		if token.ExpiresAt != nil {
			ttl := time.Until(*token.ExpiresAt)
			if ttl <= 0 {
				return fmt.Errorf("token already expired")
			}
			entry = entry.WithTTL(ttl)
		}

		return txn.SetEntry(entry)
	})
}

func (r *TokenRepository) Delete(ctx context.Context, hash string) error {
	return r.db.Update(func(txn *badgerdb.Txn) error {
		_, err := txn.Get(tokenKey(hash))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("token not found")
		}
		if err != nil {
			return err
		}
		return txn.Delete(tokenKey(hash))
	})
}

func (r *TokenRepository) DeleteByUserId(ctx context.Context, userId string) error {
	return r.db.Update(func(txn *badgerdb.Txn) error {
		opts := badgerdb.DefaultIteratorOptions
		opts.Prefix = []byte(tokenPrefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		var keys [][]byte

		for it.Rewind(); it.Valid(); it.Next() {
			var token *domain.Token
			err := it.Item().Value(func(val []byte) error {
				var err error
				token, err = r.unmarshal(val)
				return err
			})
			if err != nil {
				return err
			}

			if token.UserId == userId {
				keys = append(keys, it.Item().KeyCopy(nil))
			}
		}

		for _, key := range keys {
			if err := txn.Delete(key); err != nil {
				return err
			}
		}

		return nil
	})
}
