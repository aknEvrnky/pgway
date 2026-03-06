package csv

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"go.uber.org/zap"
)

var ErrPoolNotFound = errors.New("pool not found")

func (r *CsvRepository) getTempProxyId() string {
	r.proxyIdCounter.Add(1)
	return strconv.FormatUint(uint64(r.proxyIdCounter.Load()), 10)
}

type CsvRepository struct {
	path           string
	pools          map[string]*domain.Pool
	proxyIdCounter atomic.Uint32
}

func NewCsvRepository(path string) (*CsvRepository, error) {
	repo := &CsvRepository{
		path:  path,
		pools: make(map[string]*domain.Pool),
	}

	err := repo.load()

	if err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *CsvRepository) load() error {
	// read from csv
	entries, err := os.ReadDir(r.path)
	if err != nil {
		return fmt.Errorf("reading pools dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".csv" {
			continue
		}

		poolId := strings.TrimSuffix(entry.Name(), ".csv")
		path := filepath.Join(r.path, entry.Name())

		proxies, err := r.loadPoolFromCSV(path)
		if err != nil {
			return fmt.Errorf("pool %q: %w", poolId, err)
		}

		r.pools[poolId] = &domain.Pool{
			Id:      poolId,
			Title:   poolId,
			Tags:    []string{"default"},
			Proxies: proxies,
		}
	}

	return nil
}

func (r *CsvRepository) loadPoolFromCSV(path string) ([]*domain.Proxy, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("opening pool file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)

	reader.Comment = '#'
	reader.TrimLeadingSpace = true

	var proxies []*domain.Proxy

	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("reading csv: %w", err)
		}

		if len(record) == 0 {
			continue
		}

		proxy, err := domain.NewProxyFromURL(record[0])

		if err != nil {
			zap.L().Warn("skipping proxy",
				zap.String("path", path),
				zap.String("raw", record[0]),
				zap.Error(err),
			)

			continue
		}

		proxy.Id = r.getTempProxyId()

		proxies = append(proxies, proxy)
	}

	if len(proxies) == 0 {
		return nil, fmt.Errorf("no valid proxy found in pool %q", path)
	}

	return proxies, nil
}

func (r *CsvRepository) GetAll(ctx context.Context) ([]*domain.Pool, error) {
	results := make([]*domain.Pool, 0, len(r.pools))

	for _, p := range r.pools {
		results = append(results, p)
	}

	return results, nil
}

func (r *CsvRepository) Find(ctx context.Context, id string) (*domain.Pool, error) {
	f, ok := r.pools[id]
	if !ok {
		return nil, ErrPoolNotFound
	}
	return f, nil
}
