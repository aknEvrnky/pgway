package probes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProbeHandler_LivenessAlwaysOK(t *testing.T) {
	g := NewReadyGate(ReadyGateConfig{RequireGRPC: true}) // not serving
	h := New(":0", g).handler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", strings.TrimSpace(rec.Body.String()))
}

func TestProbeHandler_ReadinessReflectsGate(t *testing.T) {
	g := NewReadyGate(ReadyGateConfig{RequireGRPC: true})
	h := New(":0", g).handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, ReasonGRPCNotServing, strings.TrimSpace(rec.Body.String()))

	g.MarkGRPCServing(true)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, ReasonOK, strings.TrimSpace(rec.Body.String()))
}
