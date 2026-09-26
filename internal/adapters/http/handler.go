package http

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/balancer"
	"github.com/aknEvrnky/pgway/internal/platform/metrics"
	"github.com/aknEvrnky/pgway/internal/platform/tracing"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const copyBufSize = 32 * 1024

var copyBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, copyBufSize)
		return &b
	},
}

func getCopyBuf() *[]byte {
	return copyBufPool.Get().(*[]byte)
}

func putCopyBuf(b *[]byte) {
	copyBufPool.Put(b)
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
	link         ports.CPLinkStatus // optional
	maxBodyBytes int64
}

func NewHandler(app ports.Application, t ports.ProxyTransportPort, maxBodyBytes int64, link ports.CPLinkStatus) *Handler {
	return &Handler{
		app:          app,
		transport:    t,
		link:         link,
		maxBodyBytes: maxBodyBytes,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, span := tracing.Tracer().Start(r.Context(), tracing.SpanProxyRequest)
	r = r.WithContext(ctx)

	entrypointId, ok := r.Context().Value(entrypointContextKey).(contextKey)
	protocol := metrics.ProtocolHTTP
	if r.Method == http.MethodConnect {
		protocol = metrics.ProtocolConnect
	}

	ep := "unknown"
	result := metrics.ResultRejected
	defer func() {
		span.SetAttributes(
			attribute.String("entrypoint", ep),
			attribute.String("result", result),
			attribute.String("protocol", protocol),
		)
		span.End()
	}()

	if !ok {
		zap.L().Info("missing entrypoint", zap.String("ep", string(entrypointId)))
		http.Error(w, "missing entrypoint", http.StatusInternalServerError)
		metrics.RecordProxy(r.Context(), metrics.ProxyRecord{
			Entrypoint: ep,
			Protocol:   protocol,
			Result:     result,
			Duration:   time.Since(start),
		})
		return
	}
	ep = string(entrypointId)

	if h.link != nil {
		snap := h.link.Snapshot()
		if snap.Rejecting {
			w.Header().Set("X-Pgway-Reject-Reason", "cp_unreachable")
			retryAfter := snap.RetryAfterSeconds
			if retryAfter < 1 {
				retryAfter = 30
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			http.Error(w, "control plane unreachable", http.StatusServiceUnavailable)
			metrics.RecordProxy(r.Context(), metrics.ProxyRecord{
				Entrypoint: ep,
				Protocol:   protocol,
				Result:     result,
				Duration:   time.Since(start),
			})
			return
		}
	}

	// Reject oversized bodies before ExecuteFlow so balancer state is not
	// consumed for requests we will never forward. ContentLength < 0
	// (chunked / unknown) is handled later via MaxBytesReader.
	if r.Method != http.MethodConnect && h.maxBodyBytes > 0 && r.ContentLength > h.maxBodyBytes {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		metrics.RecordProxy(r.Context(), metrics.ProxyRecord{
			Entrypoint: ep,
			Protocol:   protocol,
			Result:     result,
			Duration:   time.Since(start),
		})
		return
	}

	proxy, balancerId, err := h.app.ExecuteFlow(r.Context(), ep, r)
	if err != nil {
		result = flowErrorResult(err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		metrics.RecordProxy(r.Context(), metrics.ProxyRecord{
			Entrypoint: ep,
			Protocol:   protocol,
			Result:     result,
			Duration:   time.Since(start),
		})
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

	metrics.ActiveProxyDelta(r.Context(), protocol, 1)
	// The request context is canceled on CONNECT hijack before defers run;
	// record the decrement on a cancel-safe context so the gauge drains.
	defer metrics.ActiveProxyDelta(context.WithoutCancel(r.Context()), protocol, -1)

	result = metrics.ResultOK
	if r.Method == http.MethodConnect {
		h.handleTunnel(w, r, proxy, &transferred, ep, start, &result)
		return
	}

	h.handleHTTP(w, r, proxy, &transferred, ep, start, &result)
}

func (h *Handler) handleTunnel(w http.ResponseWriter, r *http.Request, proxy *domain.Proxy, transferred *int64, ep string, start time.Time, result *string) {
	*result = metrics.ResultOK
	var bytesIn int64
	defer func() {
		metrics.RecordProxy(context.WithoutCancel(r.Context()), metrics.ProxyRecord{
			Entrypoint: ep,
			Protocol:   metrics.ProtocolConnect,
			Result:     *result,
			Duration:   time.Since(start),
			BytesIn:    bytesIn,
			BytesOut:   *transferred,
		})
	}()

	upCtx, upSpan := tracing.Tracer().Start(r.Context(), tracing.SpanProxyUpstream)
	dst, err := h.transport.Dial(upCtx, proxy, r.Host)
	upSpan.End()
	if err != nil {
		zap.L().Error("dial failed", zap.Error(err), zap.String("proxy", proxy.Addr()), zap.String("target", r.Host))
		status, msg := classifyUpstreamError(err)
		*result = resultFromHTTPStatus(status)
		http.Error(w, msg, status)
		return
	}

	defer dst.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		*result = metrics.ResultError
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	src, bufrw, err := hijacker.Hijack()
	if err != nil {
		zap.L().Error("hijack failed", zap.Error(err))
		*result = metrics.ResultError
		return
	}
	defer src.Close()

	if _, err := bufrw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		zap.L().Error("write connect ok", zap.Error(err))
		*result = metrics.ResultError
		return
	}
	if err := bufrw.Flush(); err != nil {
		zap.L().Error("flush connect ok", zap.Error(err))
		*result = metrics.ResultError
		return
	}

	g, _ := errgroup.WithContext(r.Context())

	g.Go(func() error {
		bufp := getCopyBuf()
		defer putCopyBuf(bufp)
		n, err := io.CopyBuffer(dst, bufrw, *bufp)
		bytesIn = n
		closeWrite(dst)
		return err
	})

	g.Go(func() error {
		bufp := getCopyBuf()
		defer putCopyBuf(bufp)
		n, err := io.CopyBuffer(bufrw, dst, *bufp)
		*transferred = n
		_ = bufrw.Flush()
		closeWrite(src)
		return err
	})

	if err := g.Wait(); err != nil {
		zap.L().Debug("tunnel copy", zap.Error(err))
	}
}

func (h *Handler) handleHTTP(w http.ResponseWriter, r *http.Request, proxy *domain.Proxy, transferred *int64, ep string, start time.Time, result *string) {
	r.RequestURI = ""
	h.removeHopHeaders(r.Header)

	if h.maxBodyBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, h.maxBodyBytes)
	}
	// Wrap after the size cap so the count reflects what is actually consumed.
	cr := &countingReadCloser{rc: r.Body}
	r.Body = cr

	*result = metrics.ResultOK
	defer func() {
		metrics.RecordProxy(context.WithoutCancel(r.Context()), metrics.ProxyRecord{
			Entrypoint: ep,
			Protocol:   metrics.ProtocolHTTP,
			Result:     *result,
			Duration:   time.Since(start),
			BytesIn:    cr.count(),
			BytesOut:   *transferred,
		})
	}()

	upCtx, upSpan := tracing.Tracer().Start(r.Context(), tracing.SpanProxyUpstream)
	resp, err := h.transport.RoundTrip(upCtx, proxy, r)
	upSpan.End()
	if err != nil {
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			*result = metrics.ResultRejected
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		status, msg := classifyUpstreamError(err)
		*result = resultFromHTTPStatus(status)
		http.Error(w, msg, status)
		return
	}
	defer resp.Body.Close()

	h.removeHopHeaders(resp.Header)
	h.copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	bufp := getCopyBuf()
	defer putCopyBuf(bufp)
	n, err := io.CopyBuffer(w, resp.Body, *bufp)
	*transferred = n
	if err != nil {
		zap.L().Error("copy response body", zap.Error(err))
		*result = metrics.ResultError
	}
}

type countingReadCloser struct {
	rc io.ReadCloser
	n  atomic.Int64
}

func (c *countingReadCloser) Read(p []byte) (int, error) {
	n, err := c.rc.Read(p)
	c.n.Add(int64(n))
	return n, err
}

func (c *countingReadCloser) Close() error { return c.rc.Close() }

func (c *countingReadCloser) count() int64 { return c.n.Load() }

func resultFromHTTPStatus(status int) string {
	if status == http.StatusGatewayTimeout {
		return metrics.ResultTimeout
	}
	if status == http.StatusRequestEntityTooLarge || status == http.StatusServiceUnavailable {
		return metrics.ResultRejected
	}
	return metrics.ResultError
}

// flowErrorResult classifies ExecuteFlow failures: missing configuration
// (entrypoint, flow, balancer, routing rule, empty pool) is a gateway-side
// rejection, anything else is an internal error.
func flowErrorResult(err error) string {
	switch {
	case errors.Is(err, domain.ErrNotFound),
		errors.Is(err, domain.ErrNoProxy),
		errors.Is(err, domain.ErrNoMatchingRule),
		errors.Is(err, balancer.ErrBalancerNotFound):
		return metrics.ResultRejected
	default:
		return metrics.ResultError
	}
}

func (h *Handler) removeHopHeaders(header http.Header) {
	for _, connVal := range header.Values("Connection") {
		for _, token := range strings.Split(connVal, ",") {
			token = strings.TrimSpace(token)
			if token != "" {
				header.Del(token)
			}
		}
	}

	hopHeaders := []string{
		"Connection", "Proxy-Connection", "Keep-Alive",
		"Proxy-Authenticate", "Proxy-Authorization",
		"Te", "Trailer", "Transfer-Encoding", "Upgrade",
	}
	for _, name := range hopHeaders {
		header.Del(name)
	}
}

func (h *Handler) copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
