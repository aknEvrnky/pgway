package rest

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func (a *Adapter) cors(next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(a.cfg.CORSAllowOrigins))
	for _, o := range a.cfg.CORSAllowOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = struct{}{}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *Adapter) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeError(w, http.StatusUnauthorized, "missing or invalid authorization")
			return
		}

		principal, err := a.authenticator.Authenticate(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		if principal.Kind() != domain.PrincipalKindUser {
			writeError(w, http.StatusForbidden, "user principal required")
			return
		}

		ctx := ports.ContextWithPrincipal(r.Context(), principal)
		ctx = ports.ContextWithToken(ctx, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// bearerToken parses an Authorization header. The auth scheme is
// case-insensitive per RFC 7235.
func bearerToken(header string) (string, bool) {
	scheme, rest, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(rest)
	return token, token != ""
}

// rateLimit returns middleware that buckets requests by keyFn. Two stages are
// installed: per-IP before auth (caps token brute-force and anonymous floods)
// and per-user after auth (caps a single credential).
func (a *Adapter) rateLimit(keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if a.cfg.RateLimitRPS <= 0 {
			return next
		}

		registry := newLimiterRegistry(rate.Limit(a.cfg.RateLimitRPS), a.cfg.RateLimitBurst)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !registry.allow(keyFn(r)) {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func userLimitKey(r *http.Request) string {
	if p, ok := ports.PrincipalFromContext(r.Context()); ok && p.User != nil && p.User.Id != "" {
		return "user:" + p.User.Id
	}
	return ipLimitKey(r)
}

func ipLimitKey(r *http.Request) string {
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if host != "" {
		return "ip:" + host
	}
	return "anon"
}

func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				zap.L().Error("panic recovered", zap.Any("error", err), zap.String("path", r.URL.Path))
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		zap.L().Info("http request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", sw.status),
			zap.Duration("duration", time.Since(start)),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
