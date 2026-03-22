package http

import (
	"io"
	"net/http"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

type Handler struct {
	app       ports.Application
	transport ports.ProxyTransportPort
}

func NewHandler(app ports.Application, t ports.ProxyTransportPort) *Handler {
	return &Handler{
		app:       app,
		transport: t,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	entrypointId, ok := r.Context().Value(entrypointContextKey).(contextKey)
	if !ok {
		zap.L().Info("missing entrypoint", zap.String("ep", string(entrypointId)))

		http.Error(w, "missing entrypoint", http.StatusInternalServerError)
		return
	}

	proxy, balancerId, err := h.app.ExecuteFlow(r.Context(), string(entrypointId), r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	zap.L().Info("using proxy", zap.String("proxy", proxy.Id))

	// todo: handle byte calculation
	defer h.app.Release(r.Context(), balancerId, domain.BalancerResult{
		ProxyId: proxy.Id,
		Bytes:   0,
	})

	if r.Method == http.MethodConnect {
		h.handleTunnel(w, r, proxy)
		return
	}

	h.handleHTTP(w, r, proxy)
}

func (h *Handler) handleTunnel(w http.ResponseWriter, r *http.Request, proxy *domain.Proxy) {
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

	go io.Copy(dst, src)
	io.Copy(src, dst)
}

// HTTP — direkt forward
func (h *Handler) handleHTTP(w http.ResponseWriter, r *http.Request, proxy *domain.Proxy) {
	// Hop-by-hop header'ları temizle
	r.RequestURI = ""
	h.removeHopHeaders(r.Header)

	resp, err := h.transport.RoundTrip(r.Context(), proxy, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	h.removeHopHeaders(resp.Header)
	h.copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (h *Handler) removeHopHeaders(header http.Header) {
	hopHeaders := []string{
		"Connection", "Proxy-Connection", "Keep-Alive",
		"Proxy-Authenticate", "Proxy-Authorization",
		"Te", "Trailers", "Transfer-Encoding", "Upgrade",
	}
	for _, hh := range hopHeaders {
		header.Del(hh)
	}
}

func (h *Handler) copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}
