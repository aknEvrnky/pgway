package http

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	dialErr        error
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
	if t.dialErr != nil {
		return nil, t.dialErr
	}
	c1, c2 := net.Pipe()
	_ = c2.Close()
	return c1, nil
}

func withEntrypoint(r *http.Request, id string) *http.Request {
	ctx := context.WithValue(r.Context(), entrypointContextKey, contextKey(id))
	return r.WithContext(ctx)
}

func TestHandler_RoundTripTimeout_Returns504(t *testing.T) {
	api := &handlerFakeAPI{proxy: &domain.Proxy{Id: "p1"}, balancerID: "lb1"}
	tr := &handlerFakeTransport{roundTripErr: timeoutError{}}
	h := NewHandler(api, tr, 0, nil)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req = withEntrypoint(req, "ep1")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusGatewayTimeout, rec.Code)
	assert.Contains(t, rec.Body.String(), "gateway timeout")
	assert.Equal(t, int32(1), tr.roundTripCalls.Load())
}

func TestHandler_DialRefused_Returns502(t *testing.T) {
	api := &handlerFakeAPI{proxy: &domain.Proxy{Id: "p1", Host: "127.0.0.1", Port: 1}, balancerID: "lb1"}
	tr := &handlerFakeTransport{dialErr: syscall.ECONNREFUSED}
	h := NewHandler(api, tr, 0, nil)

	req := httptest.NewRequest(http.MethodConnect, "http://example.com:443", nil)
	req.Host = "example.com:443"
	req = withEntrypoint(req, "ep1")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Contains(t, rec.Body.String(), "connection refused")
	assert.Equal(t, int32(1), tr.dialCalls.Load())
}

func TestHandler_ContentLengthOverLimit_Returns413(t *testing.T) {
	api := &handlerFakeAPI{proxy: &domain.Proxy{Id: "p1"}, balancerID: "lb1"}
	tr := &handlerFakeTransport{}
	h := NewHandler(api, tr, 100, nil)

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
	h := NewHandler(api, tr, 64, nil)

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
	h := NewHandler(api, tr, 1024, nil)

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
	h := NewHandler(api, tr, 0, nil)

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
	h := NewHandler(api, tr, 10, nil)

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

type closeWriteStub struct {
	net.Conn
	calls atomic.Int32
}

func (c *closeWriteStub) CloseWrite() error {
	c.calls.Add(1)
	if tc, ok := c.Conn.(*net.TCPConn); ok {
		return tc.CloseWrite()
	}
	return nil
}

func TestCloseWrite_Soft(t *testing.T) {
	t.Parallel()

	t.Run("no-op when unsupported", func(t *testing.T) {
		a, b := net.Pipe()
		t.Cleanup(func() { _ = a.Close(); _ = b.Close() })
		assert.NotPanics(t, func() { closeWrite(a) })
	})

	t.Run("calls CloseWrite when present", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = ln.Close() })

		accepted := make(chan net.Conn, 1)
		go func() {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			accepted <- c
		}()

		client, err := net.Dial("tcp", ln.Addr().String())
		require.NoError(t, err)
		t.Cleanup(func() { _ = client.Close() })

		server := <-accepted
		t.Cleanup(func() { _ = server.Close() })

		stub := &closeWriteStub{Conn: server}
		closeWrite(stub)
		assert.Equal(t, int32(1), stub.calls.Load())
	})
}

type hijackResponse struct {
	header http.Header
	code   int
	conn   net.Conn
}

func (h *hijackResponse) Header() http.Header {
	if h.header == nil {
		h.header = make(http.Header)
	}
	return h.header
}

func (h *hijackResponse) Write(b []byte) (int, error) { return len(b), nil }

func (h *hijackResponse) WriteHeader(code int) { h.code = code }

func (h *hijackResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return h.conn, bufio.NewReadWriter(bufio.NewReader(h.conn), bufio.NewWriter(h.conn)), nil
}

type tunnelFakeTransport struct {
	dst       net.Conn
	dialCalls atomic.Int32
}

func (t *tunnelFakeTransport) RoundTrip(context.Context, *domain.Proxy, *http.Request) (*http.Response, error) {
	return nil, nil
}

func (t *tunnelFakeTransport) Dial(context.Context, *domain.Proxy, string) (net.Conn, error) {
	t.dialCalls.Add(1)
	return t.dst, nil
}

func TestHandler_CONNECT_HalfCloseAndTransfer(t *testing.T) {
	// Upstream echo server (what Dial returns a connection to).
	upLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = upLn.Close() })

	go func() {
		c, err := upLn.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 64)
		n, err := c.Read(buf)
		if err != nil {
			return
		}
		_, _ = c.Write(buf[:n])
		if tc, ok := c.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()

	upConn, err := net.Dial("tcp", upLn.Addr().String())
	require.NoError(t, err)
	upStub := &closeWriteStub{Conn: upConn}

	// Client side of the hijacked connection.
	clientLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = clientLn.Close() })

	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := clientLn.Accept()
		if err != nil {
			return
		}
		accepted <- c
	}()

	peer, err := net.Dial("tcp", clientLn.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = peer.Close() })

	hijacked := <-accepted
	clientStub := &closeWriteStub{Conn: hijacked}

	api := &handlerFakeAPI{proxy: &domain.Proxy{Id: "p1"}, balancerID: "lb1"}
	tr := &tunnelFakeTransport{dst: upStub}
	h := NewHandler(api, tr, 0, nil)

	req := httptest.NewRequest(http.MethodConnect, "http://example.com:443", nil)
	req.Host = "example.com:443"
	req = withEntrypoint(req, "ep1")

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.ServeHTTP(&hijackResponse{conn: clientStub}, req)
	}()

	// CONNECT 200 must land on the wire before tunnel bytes.
	require.NoError(t, peer.SetReadDeadline(time.Now().Add(2*time.Second)))
	br := bufio.NewReader(peer)
	status, err := br.ReadString('\n')
	require.NoError(t, err)
	assert.Contains(t, status, "200")
	// Skip headers until blank line.
	for {
		line, err := br.ReadString('\n')
		require.NoError(t, err)
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	payload := []byte("ping-tunnel")
	_, err = peer.Write(payload)
	require.NoError(t, err)

	got := make([]byte, len(payload))
	require.NoError(t, peer.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, err = io.ReadFull(br, got)
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	_ = peer.Close()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("tunnel did not finish after client close")
	}

	assert.GreaterOrEqual(t, upStub.calls.Load(), int32(1))
	assert.GreaterOrEqual(t, clientStub.calls.Load(), int32(1))
	assert.Equal(t, int32(1), tr.dialCalls.Load())
}

func TestHandler_CONNECT_200ReachesRealHTTPClient(t *testing.T) {
	// Upstream blackhole (accepts dial, holds connection).
	upLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = upLn.Close() })
	go func() {
		c, err := upLn.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		time.Sleep(2 * time.Second)
	}()

	upConn, err := net.Dial("tcp", upLn.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = upConn.Close() })

	api := &handlerFakeAPI{proxy: &domain.Proxy{Id: "p1"}, balancerID: "lb1"}
	tr := &tunnelFakeTransport{dst: upConn}
	h := NewHandler(api, tr, 0, nil)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = withEntrypoint(r, "ep1")
		h.ServeHTTP(w, r)
	})}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	conn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	_, err = io.WriteString(conn, "CONNECT example.com:443 HTTP/1.1\r\nHost: example.com:443\r\n\r\n")
	require.NoError(t, err)

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(statusLine, "HTTP/1.1 200"), "got %q", statusLine)
}

func TestHandler_removeHopHeaders_StripsConnectionTokens(t *testing.T) {
	h := NewHandler(&handlerFakeAPI{}, &handlerFakeTransport{}, 0, nil)
	hdr := make(http.Header)
	hdr.Set("Connection", "keep-alive, X-Foo")
	hdr.Set("Keep-Alive", "timeout=5")
	hdr.Set("X-Foo", "bar")
	hdr.Set("X-Keep", "yes")

	h.removeHopHeaders(hdr)

	assert.Empty(t, hdr.Get("Connection"))
	assert.Empty(t, hdr.Get("Keep-Alive"))
	assert.Empty(t, hdr.Get("X-Foo"), "header named in Connection must be stripped")
	assert.Equal(t, "yes", hdr.Get("X-Keep"))
}

type staticLinkStatus struct {
	snap ports.CPLinkSnapshot
}

func (s staticLinkStatus) Snapshot() ports.CPLinkSnapshot { return s.snap }

func TestHandler_FailClosedUnreachable_RejectsNewRequest(t *testing.T) {
	api := &handlerFakeAPI{proxy: &domain.Proxy{Id: "p1"}, balancerID: "lb1"}
	tr := &handlerFakeTransport{}
	link := staticLinkStatus{snap: ports.CPLinkSnapshot{
		State:             ports.CPLinkUnreachable,
		Strategy:          ports.CPDisconnectFailClosed,
		Rejecting:         true,
		RetryAfterSeconds: 45,
	}}
	h := NewHandler(api, tr, 0, link)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req = withEntrypoint(req, "ep1")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "control plane unreachable")
	assert.Equal(t, "cp_unreachable", rec.Header().Get("X-Pgway-Reject-Reason"))
	assert.Equal(t, "45", rec.Header().Get("Retry-After"))
	assert.Equal(t, int32(0), tr.roundTripCalls.Load())
}

func TestHandler_FailOpenUnreachable_AllowsTraffic(t *testing.T) {
	api := &handlerFakeAPI{proxy: &domain.Proxy{Id: "p1"}, balancerID: "lb1"}
	tr := &handlerFakeTransport{}
	link := staticLinkStatus{snap: ports.CPLinkSnapshot{
		State:     ports.CPLinkUnreachable,
		Strategy:  ports.CPDisconnectFailOpen,
		Rejecting: false,
	}}
	h := NewHandler(api, tr, 0, link)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req = withEntrypoint(req, "ep1")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int32(1), tr.roundTripCalls.Load())
}
