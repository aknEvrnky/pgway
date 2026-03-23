package rest

import "net/http"

func (a *Adapter) routes() http.Handler {
	mux := http.NewServeMux()

	// Proxy routes
	mux.HandleFunc("GET /api/v1/proxies", a.listProxies)
	mux.HandleFunc("GET /api/v1/proxies/{name}", a.getProxy)
	mux.HandleFunc("POST /api/v1/proxies", a.applyProxy)
	mux.HandleFunc("DELETE /api/v1/proxies/{name}", a.deleteProxy)

	// Middleware chain
	var handler http.Handler = mux
	handler = cors(handler)
	handler = recovery(handler)
	handler = logging(handler)

	return handler
}
