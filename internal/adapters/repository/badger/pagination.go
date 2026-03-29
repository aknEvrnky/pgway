package badger

import (
	"bytes"
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
	prefixBytes := []byte(prefix)

	if predicate == nil {
		// No filter: keys-only count, then paginated value fetch
		countOpts := badgerdb.DefaultIteratorOptions
		countOpts.Prefix = prefixBytes
		countOpts.PrefetchValues = false
		countIt := txn.NewIterator(countOpts)
		for countIt.Rewind(); countIt.Valid(); countIt.Next() {
			result.TotalCount++
		}
		countIt.Close()

		opts := badgerdb.DefaultIteratorOptions
		opts.Prefix = prefixBytes
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

	// Filtered: single pass — count + collect in one iterator scan.
	// Items before cursor contribute only to TotalCount.
	// Items from cursor onward contribute to both TotalCount and page collection.
	opts := badgerdb.DefaultIteratorOptions
	opts.Prefix = prefixBytes
	it := txn.NewIterator(opts)
	defer it.Close()

	cursorKey := []byte(prefix + params.Cursor)
	reachedCursor := params.Cursor == ""
	returnAll := params.PageSize == 0
	collected := 0
	pageFull := false

	for it.Rewind(); it.Valid(); it.Next() {
		var item *T
		err := it.Item().Value(func(val []byte) error {
			var err error
			item, err = unmarshal(val)
			return err
		})
		if err != nil {
			return result, err
		}

		if !predicate(item) {
			continue
		}

		result.TotalCount++

		if !reachedCursor {
			key := it.Item().Key()
			if bytes.Compare(key, cursorKey) >= 0 {
				reachedCursor = true
			} else {
				continue
			}
		}

		if pageFull {
			continue
		}

		if !returnAll && collected >= params.PageSize {
			key := it.Item().Key()
			result.NextCursor = string(key[len(prefix):])
			pageFull = true
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
