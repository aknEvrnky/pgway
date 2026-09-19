package agenthost

import (
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLinkState_ANDStaleUnreachable(t *testing.T) {
	clock := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	ls := NewLinkState(LinkStateConfig{
		Strategy:             ports.CPDisconnectFailClosed,
		UnreachableThreshold: 30 * time.Second,
	})
	ls.now = func() time.Time { return clock }

	_ = ls.Snapshot() // arm bothStaleSince
	clock = clock.Add(30 * time.Second)
	snap := ls.Snapshot()
	assert.Equal(t, ports.CPLinkUnreachable, snap.State)
	assert.True(t, snap.Rejecting)
}

func TestLinkState_OneProofKeepsReachable(t *testing.T) {
	clock := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	ls := NewLinkState(LinkStateConfig{
		Strategy:             ports.CPDisconnectFailOpen,
		UnreachableThreshold: 30 * time.Second,
	})
	ls.now = func() time.Time { return clock }

	ls.MarkWatchUp()
	clock = clock.Add(time.Hour) // HB never touched → stale, watch up
	snap := ls.Snapshot()
	assert.Equal(t, ports.CPLinkDegraded, snap.State)
	assert.False(t, snap.Rejecting)

	ls.MarkWatchDown()
	ls.TouchHeartbeat()
	snap = ls.Snapshot()
	assert.Equal(t, ports.CPLinkDegraded, snap.State)

	// Both stale only after HB ages out while watch stays down.
	clock = clock.Add(31 * time.Second)
	snap = ls.Snapshot()
	assert.Equal(t, ports.CPLinkDegraded, snap.State) // both stale just started

	clock = clock.Add(30 * time.Second)
	snap = ls.Snapshot()
	assert.Equal(t, ports.CPLinkUnreachable, snap.State)
	assert.False(t, snap.Rejecting) // fail_open
}

func TestLinkState_RecoverThreshold(t *testing.T) {
	clock := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	ls := NewLinkState(LinkStateConfig{
		Strategy:             ports.CPDisconnectFailClosed,
		UnreachableThreshold: 10 * time.Second,
		RecoverThreshold:     5 * time.Second,
	})
	ls.now = func() time.Time { return clock }

	_ = ls.Snapshot() // arm bothStaleSince
	clock = clock.Add(10 * time.Second)
	require.Equal(t, ports.CPLinkUnreachable, ls.Snapshot().State)

	ls.TouchHeartbeat()
	ls.MarkWatchUp()
	// Still unreachable during recover hysteresis.
	assert.Equal(t, ports.CPLinkUnreachable, ls.Snapshot().State)
	assert.True(t, ls.Snapshot().Rejecting)

	clock = clock.Add(5 * time.Second)
	snap := ls.Snapshot()
	assert.Equal(t, ports.CPLinkConnected, snap.State)
	assert.False(t, snap.Rejecting)
}

func TestLinkState_ImmediateRecoverDefault(t *testing.T) {
	clock := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	ls := NewLinkState(LinkStateConfig{
		Strategy:             ports.CPDisconnectFailClosed,
		UnreachableThreshold: 10 * time.Second,
		RecoverThreshold:     0,
	})
	ls.now = func() time.Time { return clock }

	_ = ls.Snapshot()
	clock = clock.Add(10 * time.Second)
	require.Equal(t, ports.CPLinkUnreachable, ls.Snapshot().State)

	ls.TouchHeartbeat()
	ls.MarkWatchUp()
	assert.Equal(t, ports.CPLinkConnected, ls.Snapshot().State)
}

func TestLinkState_LogsStateTransitions(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	clock := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	ls := NewLinkState(LinkStateConfig{
		Strategy:             ports.CPDisconnectFailClosed,
		UnreachableThreshold: 10 * time.Second,
		Log:                  zap.New(core),
	})
	ls.now = func() time.Time { return clock }

	_ = ls.Snapshot() // init → degraded
	clock = clock.Add(10 * time.Second)
	_ = ls.Snapshot() // degraded → unreachable

	ls.TouchHeartbeat()
	ls.MarkWatchUp() // unreachable → connected

	entries := logs.FilterMessage("cp link state changed").All()
	require.GreaterOrEqual(t, len(entries), 3)

	var transitions [][2]string
	for _, e := range entries {
		from, _ := e.ContextMap()["from"].(string)
		to, _ := e.ContextMap()["to"].(string)
		transitions = append(transitions, [2]string{from, to})
	}
	assert.Contains(t, transitions, [2]string{"init", ports.CPLinkDegraded})
	assert.Contains(t, transitions, [2]string{ports.CPLinkDegraded, ports.CPLinkUnreachable})
	assert.Contains(t, transitions, [2]string{ports.CPLinkUnreachable, ports.CPLinkConnected})

	last := entries[len(entries)-1]
	assert.Equal(t, ports.CPDisconnectFailClosed, last.ContextMap()["strategy"])
	assert.Equal(t, false, last.ContextMap()["rejecting"])
}

func TestLinkState_RetryAfterMatchesThreshold(t *testing.T) {
	ls := NewLinkState(LinkStateConfig{
		Strategy:             ports.CPDisconnectFailClosed,
		UnreachableThreshold: 45 * time.Second,
	})
	clock := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	ls.now = func() time.Time { return clock }

	_ = ls.Snapshot()
	clock = clock.Add(45 * time.Second)
	snap := ls.Snapshot()
	require.Equal(t, ports.CPLinkUnreachable, snap.State)
	assert.Equal(t, 45, snap.RetryAfterSeconds)
	assert.True(t, snap.Rejecting)
}
