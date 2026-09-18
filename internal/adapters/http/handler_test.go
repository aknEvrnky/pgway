package http

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/stretchr/testify/assert"
)

type handlerFakeAPI struct {
	executeCalls atomic.Int32
	proxy        *domain.Proxy
	balancerID   string
	executeErr   error
}

func (f *handlerFakeAPI) EntryPoints(_ context.Context) ([]*domain.Entrypoint, error) {
	return nil, nil
}
func (f *handlerFakeAPI) Bootstrap(_ context.Context) error { return nil }
func (f *handlerFakeAPI) ExecuteFlow(_ context.Context, _ string, _ *http.Request) (*domain.Proxy, string, error) {
	f.executeCalls.Add(1)
	if f.executeErr != nil {
		return nil, "", f.executeErr
	}
	return f.proxy, f.balancerID, nil
}
func (f *handlerFakeAPI) Release(_ context.Context, _ string, _ domain.BalancerResult) error {
	return nil
}
func (f *handlerFakeAPI) HandleEvent(_ context.Context, _ ports.ChangeEvent) error { return nil }

type handlerFakeTransport struct {
	roundTripCalls atomic.Int32
	dialCalls      atomic.Int32
	lastBody       []byte
	roundTripErr   error
}

func (t *handlerFakeTransport) RoundTrip(_ context.Context, _ *domain.Proxy, r *http.Request) (*http.Response, error) {
	t.roundTripCalls.Add(1)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	t.lastBody = body
	if t.roundTripErr != nil {
		return nil, t.roundTripErr
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("ok")),
	}, nil
}

func (t *handlerFakeTransport) Dial(_ context.Context, _ *domain.Proxy, _ string) (net.Conn, error) {
	t.dialCalls.Add(1)
	c1, c2 := net.Pipe()
	_ = c2.Close()
	return c1, nil
}

func withEntrypoint(r *http.Request, id string) *http.Request {
	ctx := context.WithValue(r.Context(), entrypointContextKey, contextKey(id))
	return r.WithContext(ctx)
}

func TestHandler_ContentLengthOverLimit_Returns413(t *testing.T) {
	api := &handlerFakeAPI{proxy: &domain.Proxy{Id: "p1"}, balancerID: "lb1"}
	tr := &handlerFakeTransport{}
	h := NewHandler(api, tr, 100)

	req := httptest.NewRequest(http.MethodPost, "http://example.com/", nil)
	req.ContentLength = 101
	req = withEntrypoint(req, "ep1")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Equal(t, int32(0), api.executeCalls.Load())
	assert.Equal(t, int32(0), tr.roundTripCalls.Load())
}

func TestHandler_BodyStreamOverLimit_Returns413(t *testing.T) {
	api := &handlerFakeAPI{proxy: &domain.Proxy{Id: "p1"}, balancerID: "lb1"}
	tr := &handlerFakeTransport{}
	h := NewHandler(api, tr, 64)

	body := bytes.Repeat([]byte("x"), 128)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/", bytes.NewReader(body))
	// Force unknown length so early CL check is skipped.
	req.ContentLength = -1
	req = withEntrypoint(req, "ep1")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Equal(t, int32(1), api.executeCalls.Load())
	assert.Equal(t, int32(1), tr.roundTripCalls.Load())
}

func TestHandler_UnderLimit_Forwards(t *testing.T) {
	api := &handlerFakeAPI{proxy: &domain.Proxy{Id: "p1"}, balancerID: "lb1"}
	tr := &handlerFakeTransport{}
	h := NewHandler(api, tr, 1024)

	req := httptest.NewRequest(http.MethodPost, "http://example.com/", strings.NewReader("hello"))
	req = withEntrypoint(req, "ep1")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
	assert.Equal(t, []byte("hello"), tr.lastBody)
	assert.Equal(t, int32(1), tr.roundTripCalls.Load())
}

func TestHandler_ZeroLimit_DisablesCap(t *testing.T) {
	api := &handlerFakeAPI{proxy: &domain.Proxy{Id: "p1"}, balancerID: "lb1"}
	tr := &handlerFakeTransport{}
	h := NewHandler(api, tr, 0)

	body := bytes.Repeat([]byte("y"), 200)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/", bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req = withEntrypoint(req, "ep1")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, body, tr.lastBody)
}

func TestHandler_CONNECT_IgnoresBodyLimit(t *testing.T) {
	api := &handlerFakeAPI{proxy: &domain.Proxy{Id: "p1"}, balancerID: "lb1"}
	tr := &handlerFakeTransport{}
	h := NewHandler(api, tr, 10)

	req := httptest.NewRequest(http.MethodConnect, "http://example.com:443", nil)
	req.Host = "example.com:443"
	req.ContentLength = 999
	req = withEntrypoint(req, "ep1")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	// Hijack is required for tunnel; ResponseRecorder does not support it,
	// so we expect 500 after ExecuteFlow — but never 413 from body limit.
	assert.NotEqual(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Equal(t, int32(1), api.executeCalls.Load())
	assert.Equal(t, int32(0), tr.roundTripCalls.Load())
}
