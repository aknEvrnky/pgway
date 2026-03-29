package badger

import (
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	badgerdb "github.com/dgraph-io/badger/v4"
)

type unmarshalFunc[T any] func([]byte) (*T, error)

func listWithCursor[T any](
	txn *badgerdb.Txn,
	prefix string,
	params domain.ListParams,
	unmarshal unmarshalFunc[T],
) (domain.ListResult[T], error) {
	result := domain.ListResult[T]{}

	// Count total (keys-only scan)
	countOpts := badgerdb.DefaultIteratorOptions
	countOpts.Prefix = []byte(prefix)
	countOpts.PrefetchValues = false
	countIt := txn.NewIterator(countOpts)
	for countIt.Rewind(); countIt.Valid(); countIt.Next() {
		result.TotalCount++
	}
	countIt.Close()

	// Paginated fetch
	opts := badgerdb.DefaultIteratorOptions
	opts.Prefix = []byte(prefix)
	it := txn.NewIterator(opts)
	defer it.Close()

	if params.Cursor != "" {
		it.Seek([]byte(prefix + params.Cursor))
	} else {
		it.Rewind()
	}

	returnAll := params.PageSize == 0
	collected := 0

	for ; it.Valid(); it.Next() {
		if !returnAll && collected >= params.PageSize {
			key := it.Item().Key()
			result.NextCursor = string(key[len(prefix):])
			break
		}

		err := it.Item().Value(func(val []byte) error {
			item, err := unmarshal(val)
			if err != nil {
				return err
			}
			result.Items = append(result.Items, item)
			return nil
		})
		if err != nil {
			return result, err
		}
		collected++
	}

	return result, nil
}
