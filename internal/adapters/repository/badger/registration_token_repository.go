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
	registrationTokenStorageVersion = "v1"
	registrationTokenKind           = "RegistrationToken"
	registrationTokenPrefix         = "registration_tokens:"
)

type RegistrationTokenRepository struct {
	db *badgerdb.DB
}

func NewRegistrationTokenRepository(db *badgerdb.DB) *RegistrationTokenRepository {
	return &RegistrationTokenRepository{db: db}
}

func registrationTokenKey(hash string) []byte {
	return []byte(registrationTokenPrefix + hash)
}

func (r *RegistrationTokenRepository) marshal(token *domain.RegistrationToken) ([]byte, error) {
	return json.Marshal(StoredResource[domain.RegistrationToken]{
		StorageVersion: registrationTokenStorageVersion,
		Kind:           registrationTokenKind,
		Spec:           *token,
	})
}

func (r *RegistrationTokenRepository) unmarshal(data []byte) (*domain.RegistrationToken, error) {
	stored, err := unmarshal[domain.RegistrationToken](data)
	if err != nil {
		return nil, err
	}
	return &stored.Spec, nil
}

func (r *RegistrationTokenRepository) Save(ctx context.Context, token *domain.RegistrationToken) error {
	if token.ExpiresAt == nil {
		return domain.ErrTokenMustExpire
	}

	data, err := r.marshal(token)
	if err != nil {
		return fmt.Errorf("marshall registration token: %w", err)
	}

	return r.db.Update(func(txn *badgerdb.Txn) error {
		entry := badgerdb.NewEntry(registrationTokenKey(token.Hash), data)

		// badger evicts the record automatically once the token expires
		ttl := time.Until(*token.ExpiresAt)
		if ttl <= 0 {
			return fmt.Errorf("token already expired")
		}
		entry = entry.WithTTL(ttl)

		return txn.SetEntry(entry)
	})
}

func (r *RegistrationTokenRepository) Consume(ctx context.Context, hash string) (*domain.RegistrationToken, error) {
	var token *domain.RegistrationToken

	err := r.db.Update(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(registrationTokenKey(hash))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("registration token not found")
		}
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			token, err = r.unmarshal(val)
			return err
		})

		if err != nil {
			return err
		}

		return txn.Delete(registrationTokenKey(hash))
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}
