package badger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	badgerdb "github.com/dgraph-io/badger/v4"
)

const (
	userStorageVersion = "v1"
	userKind           = "User"
	userPrefix         = "users:"
)

type UserRepository struct {
	db *badgerdb.DB
}

func NewUserRepository(db *badgerdb.DB) *UserRepository {
	return &UserRepository{db: db}
}

func userKey(id string) []byte {
	return []byte(userPrefix + id)
}

func (r *UserRepository) marshal(user *domain.User) ([]byte, error) {
	return json.Marshal(StoredResource[domain.User]{
		StorageVersion: userStorageVersion,
		Kind:           userKind,
		Spec:           *user,
	})
}

func (r *UserRepository) unmarshal(data []byte) (*domain.User, error) {
	stored, err := unmarshal[domain.User](data)
	if err != nil {
		return nil, err
	}
	return &stored.Spec, nil
}

func (r *UserRepository) List(ctx context.Context, params domain.ListParams, filter domain.UserFilter) (domain.ListResult[domain.User], error) {
	predicate := buildUserPredicate(filter)
	var result domain.ListResult[domain.User]
	err := r.db.View(func(txn *badgerdb.Txn) error {
		var err error
		result, err = listWithCursor(txn, userPrefix, params, r.unmarshal, predicate)
		return err
	})
	return result, err
}

func buildUserPredicate(f domain.UserFilter) func(*domain.User) bool {
	if f.Search == "" && f.Role == "" {
		return nil
	}
	return func(u *domain.User) bool {
		if f.Search != "" && !containsFold(u.Id, f.Search) {
			return false
		}
		if f.Role != "" && string(u.Role) != f.Role {
			return false
		}
		return true
	}
}

func (r *UserRepository) Find(ctx context.Context, id string) (*domain.User, error) {
	var user *domain.User

	err := r.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(userKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("user %q not found", id)
		}

		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			user, err = r.unmarshal(val)
			return err
		})
	})

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	count := 0

	err := r.db.View(func(txn *badgerdb.Txn) error {
		opts := badgerdb.DefaultIteratorOptions
		opts.Prefix = []byte(userPrefix)
		opts.PrefetchValues = false

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})

	return count, err
}

func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
	data, err := r.marshal(user)
	if err != nil {
		return fmt.Errorf("marshall user %q: %w", user.Id, err)
	}

	return r.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Set(userKey(user.Id), data)
	})
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	return r.db.Update(func(txn *badgerdb.Txn) error {
		_, err := txn.Get(userKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("user %q not found", id)
		}
		if err != nil {
			return err
		}
		return txn.Delete(userKey(id))
	})
}
