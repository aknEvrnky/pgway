package rest

import "net/http"

func (a *Adapter) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/auth/login", a.login)
	mux.HandleFunc("POST /api/v1/auth/logout", a.logout)
	mux.HandleFunc("GET /api/v1/auth/me", a.me)

	mux.HandleFunc("GET /api/v1/proxies", a.listProxies)
	mux.HandleFunc("GET /api/v1/proxies/{name}", a.getProxy)
	mux.HandleFunc("POST /api/v1/proxies", a.applyProxy)
	mux.HandleFunc("DELETE /api/v1/proxies/{name}", a.deleteProxy)

	var handler http.Handler = mux
	// Order matters: the per-user bucket runs post-auth, the per-IP bucket
	// pre-auth so token brute-force is capped before authentication.
	handler = a.rateLimit(userLimitKey)(handler)
	handler = a.auth(handler)
	handler = a.rateLimit(ipLimitKey)(handler)
	handler = a.cors(handler)
	handler = recovery(handler)
	handler = logging(handler)

	return handler
}
