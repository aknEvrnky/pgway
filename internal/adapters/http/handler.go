package http

import (
	"errors"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const copyBufSize = 32 * 1024

var copyBufPool = sync.Pool{
	New: func() any {
		return make([]byte, copyBufSize)
	},
}

type closeWriter interface {
	CloseWrite() error
}

// closeWrite half-closes c when supported; otherwise it is a no-op.
func closeWrite(c net.Conn) {
	if cw, ok := c.(closeWriter); ok {
		_ = cw.CloseWrite()
	}
}

type Handler struct {
	app          ports.Application
	transport    ports.ProxyTransportPort
	maxBodyBytes int64
}

func NewHandler(app ports.Application, t ports.ProxyTransportPort, maxBodyBytes int64) *Handler {
	return &Handler{
		app:          app,
		transport:    t,
		maxBodyBytes: maxBodyBytes,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	entrypointId, ok := r.Context().Value(entrypointContextKey).(contextKey)
	if !ok {
		zap.L().Info("missing entrypoint", zap.String("ep", string(entrypointId)))

		http.Error(w, "missing entrypoint", http.StatusInternalServerError)
		return
	}

	// Reject oversized bodies before ExecuteFlow so balancer state is not
	// consumed for requests we will never forward. ContentLength < 0
	// (chunked / unknown) is handled later via MaxBytesReader.
	if r.Method != http.MethodConnect && h.maxBodyBytes > 0 && r.ContentLength > h.maxBodyBytes {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	proxy, balancerId, err := h.app.ExecuteFlow(r.Context(), string(entrypointId), r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	zap.L().Info("using proxy", zap.String("proxy", proxy.Id))

	var transferred int64
	defer func() {
		h.app.Release(r.Context(), balancerId, domain.BalancerResult{
			ProxyId: proxy.Id,
			Bytes:   transferred,
		})
	}()

	if r.Method == http.MethodConnect {
		h.handleTunnel(w, r, proxy, &transferred)
		return
	}

	h.handleHTTP(w, r, proxy, &transferred)
}

func (h *Handler) handleTunnel(w http.ResponseWriter, r *http.Request, proxy *domain.Proxy, transferred *int64) {
	// connect to target
	dst, err := h.transport.Dial(r.Context(), proxy, r.Host)
	if err != nil {
		zap.L().Error("dial failed", zap.Error(err), zap.String("proxy", proxy.Addr()), zap.String("target", r.Host))

		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	defer dst.Close()

	// send conn OK
	w.WriteHeader(http.StatusOK)

	// hijack the tcp conn under the client
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	src, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer src.Close()

	g, _ := errgroup.WithContext(r.Context())

	g.Go(func() error {
		buf := copyBufPool.Get().([]byte)
		defer copyBufPool.Put(buf)
		_, err := io.CopyBuffer(dst, src, buf)
		closeWrite(dst)
		return err
	})

	g.Go(func() error {
		buf := copyBufPool.Get().([]byte)
		defer copyBufPool.Put(buf)
		// Downstream only: upstream → client
		n, err := io.CopyBuffer(src, dst, buf)
		*transferred = n
		closeWrite(src)
		return err
	})

	if err := g.Wait(); err != nil {
		zap.L().Debug("tunnel copy", zap.Error(err))
	}
}

// HTTP — direkt forward
func (h *Handler) handleHTTP(w http.ResponseWriter, r *http.Request, proxy *domain.Proxy, transferred *int64) {
	// Hop-by-hop header'ları temizle
	r.RequestURI = ""
	h.removeHopHeaders(r.Header)

	if h.maxBodyBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, h.maxBodyBytes)
	}

	resp, err := h.transport.RoundTrip(r.Context(), proxy, r)
	if err != nil {
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	h.removeHopHeaders(resp.Header)
	h.copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	buf := copyBufPool.Get().([]byte)
	defer copyBufPool.Put(buf)
	n, err := io.CopyBuffer(w, resp.Body, buf)
	*transferred = n
	if err != nil {
		zap.L().Error("copy response body", zap.Error(err))
	}
}

func (h *Handler) removeHopHeaders(header http.Header) {
	hopHeaders := []string{
		"Connection", "Proxy-Connection", "Keep-Alive",
		"Proxy-Authenticate", "Proxy-Authorization",
		"Te", "Trailer", "Transfer-Encoding", "Upgrade",
	}
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func (h *Handler) copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
