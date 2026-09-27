package dashboard

import "net/http"

func (a *Adapter) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/auth/login", a.login)
	mux.HandleFunc("POST /api/v1/auth/logout", a.logout)
	mux.HandleFunc("GET /api/v1/auth/me", a.me)

	mux.HandleFunc("GET /api/v1/agents", a.listAgents)

	mux.HandleFunc("GET /api/v1/users", a.listUsers)
	mux.HandleFunc("POST /api/v1/users", a.createUser)
	mux.HandleFunc("DELETE /api/v1/users/{username}", a.deleteUser)
	mux.HandleFunc("POST /api/v1/users/{username}/password", a.changeUserPassword)

	mux.HandleFunc("GET /api/v1/proxies", a.listProxies)
	mux.HandleFunc("GET /api/v1/proxies/{name}", a.getProxy)
	mux.HandleFunc("POST /api/v1/proxies", a.applyProxy)
	mux.HandleFunc("DELETE /api/v1/proxies/{name}", a.deleteProxy)

	mux.HandleFunc("GET /api/v1/pools", a.listPools)
	mux.HandleFunc("GET /api/v1/pools/{name}", a.getPool)
	mux.HandleFunc("POST /api/v1/pools", a.applyPool)
	mux.HandleFunc("DELETE /api/v1/pools/{name}", a.deletePool)

	mux.HandleFunc("GET /api/v1/balancers", a.listBalancers)
	mux.HandleFunc("GET /api/v1/balancers/{name}", a.getBalancer)
	mux.HandleFunc("POST /api/v1/balancers", a.applyBalancer)
	mux.HandleFunc("DELETE /api/v1/balancers/{name}", a.deleteBalancer)

	mux.HandleFunc("GET /api/v1/routers", a.listRouters)
	mux.HandleFunc("GET /api/v1/routers/{name}", a.getRouter)
	mux.HandleFunc("POST /api/v1/routers", a.applyRouter)
	mux.HandleFunc("DELETE /api/v1/routers/{name}", a.deleteRouter)

	mux.HandleFunc("GET /api/v1/flows", a.listFlows)
	mux.HandleFunc("GET /api/v1/flows/{name}", a.getFlow)
	mux.HandleFunc("POST /api/v1/flows", a.applyFlow)
	mux.HandleFunc("DELETE /api/v1/flows/{name}", a.deleteFlow)

	mux.HandleFunc("GET /api/v1/entrypoints", a.listEntrypoints)
	mux.HandleFunc("GET /api/v1/entrypoints/{name}", a.getEntrypoint)
	mux.HandleFunc("POST /api/v1/entrypoints", a.applyEntrypoint)
	mux.HandleFunc("DELETE /api/v1/entrypoints/{name}", a.deleteEntrypoint)

	var handler http.Handler = mux
	if _, ok := a.contentFS(); ok {
		handler = a.withSPA(mux)
	}
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
