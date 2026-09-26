package agenthost

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type flakyRegistrar struct {
	errs  []error // returned in order; success once exhausted
	calls int
}

func (f *flakyRegistrar) Register(_ context.Context, _ string, agent domain.Agent) (*domain.Agent, string, error) {
	i := f.calls
	f.calls++
	if i < len(f.errs) {
		return nil, "", f.errs[i]
	}
	return &domain.Agent{Id: agent.Id}, "issued", nil
}

var fastRetry = RetryOptions{InitialBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}

func TestBootstrapCredentialsWithRetry_SucceedsAfterTransient(t *testing.T) {
	store := &memStore{}
	reg := &flakyRegistrar{errs: []error{errors.New("rpc error: connection refused")}}

	got, err := BootstrapCredentialsWithRetry(context.Background(), nil, store, "reg", domain.Agent{Id: "edge-1"}, reg, fastRetry)

	require.NoError(t, err)
	assert.Equal(t, "edge-1", got.AgentID)
	assert.Equal(t, "issued", got.AgentToken)
	assert.Equal(t, 2, reg.calls)
	require.NotNil(t, store.creds)
}

func TestBootstrapCredentialsWithRetry_AuthRejectedIsFatal(t *testing.T) {
	reg := &flakyRegistrar{errs: []error{fmt.Errorf("%w: token rejected", ports.ErrAgentUnauthenticated)}}

	_, err := BootstrapCredentialsWithRetry(context.Background(), nil, &memStore{}, "reg", domain.Agent{Id: "edge-1"}, reg, fastRetry)

	require.Error(t, err)
	assert.ErrorIs(t, err, ports.ErrAgentUnauthenticated)
	assert.Equal(t, 1, reg.calls)
}

func TestBootstrapCredentialsWithRetry_ConfigErrorIsFatal(t *testing.T) {
	reg := &flakyRegistrar{}

	_, err := BootstrapCredentialsWithRetry(context.Background(), nil, &memStore{}, "", domain.Agent{Id: "edge-1"}, reg, fastRetry)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBootstrapConfig)
	assert.Equal(t, 0, reg.calls)
}

func TestBootstrapCredentialsWithRetry_StateErrorIsFatal(t *testing.T) {
	store := &memStore{err: errors.New("permission denied")}
	reg := &flakyRegistrar{}

	_, err := BootstrapCredentialsWithRetry(context.Background(), nil, store, "reg", domain.Agent{Id: "edge-1"}, reg, fastRetry)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBootstrapState)
	assert.Equal(t, 0, reg.calls)
}

func TestIsPermanentBootstrapError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"config", fmt.Errorf("wrap: %w", ErrBootstrapConfig), true},
		{"state", fmt.Errorf("wrap: %w", ErrBootstrapState), true},
		{"auth", fmt.Errorf("register agent: %w: boom", ports.ErrAgentUnauthenticated), true},
		{"transient", errors.New("rpc error: connection refused"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsPermanentBootstrapError(tt.err))
		})
	}
}

func TestRetryTransient_ContextCanceledDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := RetryTransient(ctx, nil, "work", func(context.Context) error {
		calls++
		cancel()
		return errors.New("cp unreachable")
	}, func(error) bool { return false }, RetryOptions{InitialBackoff: time.Minute, MaxBackoff: time.Minute})

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
}

func TestRetryTransient_FatalStopsImmediately(t *testing.T) {
	sentinel := errors.New("fatal")
	calls := 0
	err := RetryTransient(context.Background(), nil, "work", func(context.Context) error {
		calls++
		return sentinel
	}, func(err error) bool { return errors.Is(err, sentinel) }, RetryOptions{InitialBackoff: time.Minute, MaxBackoff: time.Minute})

	require.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, calls)
}
