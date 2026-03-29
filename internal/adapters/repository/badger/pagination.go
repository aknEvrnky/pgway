package badger

import (
	"strings"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	badgerdb "github.com/dgraph-io/badger/v4"
)

type unmarshalFunc[T any] func([]byte) (*T, error)

func listWithCursor[T any](
	txn *badgerdb.Txn,
	prefix string,
	params domain.ListParams,
	unmarshal unmarshalFunc[T],
	predicate func(*T) bool,
) (domain.ListResult[T], error) {
	result := domain.ListResult[T]{}

	if predicate == nil {
		// Fast path: keys-only count when no filter
		countOpts := badgerdb.DefaultIteratorOptions
		countOpts.Prefix = []byte(prefix)
		countOpts.PrefetchValues = false
		countIt := txn.NewIterator(countOpts)
		for countIt.Rewind(); countIt.Valid(); countIt.Next() {
			result.TotalCount++
		}
		countIt.Close()
	} else {
		// Filtered count: full scan from beginning
		countOpts := badgerdb.DefaultIteratorOptions
		countOpts.Prefix = []byte(prefix)
		countIt := txn.NewIterator(countOpts)
		for countIt.Rewind(); countIt.Valid(); countIt.Next() {
			var item *T
			err := countIt.Item().Value(func(val []byte) error {
				var err error
				item, err = unmarshal(val)
				return err
			})
			if err != nil {
				countIt.Close()
				return result, err
			}
			if predicate(item) {
				result.TotalCount++
			}
		}
		countIt.Close()
	}

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

		var item *T
		err := it.Item().Value(func(val []byte) error {
			var err error
			item, err = unmarshal(val)
			return err
		})
		if err != nil {
			return result, err
		}

		if predicate != nil && !predicate(item) {
			continue
		}

		result.Items = append(result.Items, item)
		collected++
	}

	return result, nil
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func matchesLabels(actual, required map[string]string) bool {
	for k, v := range required {
		if actual[k] != v {
			return false
		}
	}
	return true
}
