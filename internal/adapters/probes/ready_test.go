package probes

import (
	"context"
	"errors"
	"testing"

	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/stretchr/testify/assert"
)

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

type errPinger struct{}

func (errPinger) Ping(context.Context) error { return errors.New("db down") }

type staticLink struct {
	snap ports.CPLinkSnapshot
}

func (s staticLink) Snapshot() ports.CPLinkSnapshot { return s.snap }

func TestReadyGate_HappyPath(t *testing.T) {
	g := NewReadyGate(ReadyGateConfig{RequireGRPC: true, Storage: okPinger{}})
	g.MarkGRPCServing(true)
	ok, reason := g.Check(context.Background())
	assert.True(t, ok)
	assert.Equal(t, ReasonOK, reason)
}

func TestReadyGate_ShuttingDown(t *testing.T) {
	g := NewReadyGate(ReadyGateConfig{})
	g.MarkShuttingDown()
	ok, reason := g.Check(context.Background())
	assert.False(t, ok)
	assert.Equal(t, ReasonShuttingDown, reason)
}

func TestReadyGate_StorageUnavailable(t *testing.T) {
	g := NewReadyGate(ReadyGateConfig{Storage: errPinger{}})
	ok, reason := g.Check(context.Background())
	assert.False(t, ok)
	assert.Equal(t, ReasonStorageUnavailable, reason)
}

func TestReadyGate_GRPCNotServing(t *testing.T) {
	g := NewReadyGate(ReadyGateConfig{RequireGRPC: true, Storage: okPinger{}})
	ok, reason := g.Check(context.Background())
	assert.False(t, ok)
	assert.Equal(t, ReasonGRPCNotServing, reason)
}

func TestReadyGate_CPUnreachableFailOpenStaysReady(t *testing.T) {
	link := staticLink{snap: ports.CPLinkSnapshot{
		State:     ports.CPLinkUnreachable,
		Strategy:  ports.CPDisconnectFailOpen,
		Rejecting: false,
	}}
	g := NewReadyGate(ReadyGateConfig{Link: link})
	ok, reason := g.Check(context.Background())
	assert.True(t, ok)
	assert.Equal(t, ReasonOK, reason)
}

func TestReadyGate_CPUnreachableFailClosedNotReady(t *testing.T) {
	link := staticLink{snap: ports.CPLinkSnapshot{
		State:     ports.CPLinkUnreachable,
		Strategy:  ports.CPDisconnectFailClosed,
		Rejecting: true,
	}}
	g := NewReadyGate(ReadyGateConfig{Link: link})
	ok, reason := g.Check(context.Background())
	assert.False(t, ok)
	assert.Equal(t, ReasonCPUnreachable, reason)
}

func TestReadyGate_CPDegradedIsReady(t *testing.T) {
	link := staticLink{snap: ports.CPLinkSnapshot{State: ports.CPLinkDegraded}}
	g := NewReadyGate(ReadyGateConfig{Link: link})
	ok, reason := g.Check(context.Background())
	assert.True(t, ok)
	assert.Equal(t, ReasonOK, reason)
}
