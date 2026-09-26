package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
)

type listProxiesResponse struct {
	Items      []*domain.Proxy `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
	TotalCount int             `json:"total_count"`
}

func (a *Adapter) listProxies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	protocol := q.Get("protocol")
	if protocol != "" && !domain.Protocol(protocol).IsValid() {
		writeError(w, http.StatusBadRequest, `protocol must be "http", "https", or "socks5"`)
		return
	}

	pageSize := defaultListPageSize
	if raw := q.Get("page_size"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "page_size must be a positive integer")
			return
		}
		pageSize = n
	}

	cursor, err := decodePageToken(q.Get("page_token"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid page_token")
		return
	}

	result, err := a.cp.ListProxies(r.Context(), domain.ListParams{
		PageSize: pageSize,
		Cursor:   cursor,
	}, domain.ProxyFilter{
		Search:   q.Get("search"),
		Protocol: protocol,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, listProxiesResponse{
		Items:      redactProxies(result.Items),
		NextCursor: encodePageToken(result.NextCursor),
		TotalCount: result.TotalCount,
	})
}

func (a *Adapter) getProxy(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	proxy, err := a.cp.GetProxy(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, redactProxy(proxy))
}

type applyProxyRequest struct {
	Metadata schema.Metadata     `json:"metadata"`
	Spec     proxyv1.ProxySpecV1 `json:"spec"`
}

func (a *Adapter) applyProxy(w http.ResponseWriter, r *http.Request) {
	var req applyProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Metadata.Name == "" {
		writeError(w, http.StatusBadRequest, "metadata.name is required")
		return
	}

	if err := req.Spec.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	proxy, err := a.cp.ApplyProxyV1(r.Context(), req.Metadata, req.Spec)
	if err != nil {
		writeCPError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, redactProxy(proxy))
}

func (a *Adapter) deleteProxy(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := a.cp.DeleteProxy(r.Context(), name); err != nil {
		writeCPError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
