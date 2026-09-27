package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
)

type listPoolsResponse struct {
	Items      []*domain.Pool `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
	TotalCount int            `json:"total_count"`
}

func (a *Adapter) listPools(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	poolType := q.Get("type")
	if poolType != "" && poolType != string(domain.PoolTypeStatic) && poolType != string(domain.PoolTypeDynamic) {
		writeError(w, http.StatusBadRequest, `type must be "static" or "dynamic"`)
		return
	}

	pageSize, err := parsePageSize(q.Get("page_size"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cursor, err := decodePageToken(q.Get("page_token"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid page_token")
		return
	}

	result, err := a.cp.ListPools(r.Context(), domain.ListParams{
		PageSize: pageSize,
		Cursor:   cursor,
	}, domain.PoolFilter{
		Search: q.Get("search"),
		Type:   poolType,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, listPoolsResponse{
		Items:      result.Items,
		NextCursor: encodePageToken(result.NextCursor),
		TotalCount: result.TotalCount,
	})
}

func (a *Adapter) getPool(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	pool, err := a.cp.GetPool(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, pool)
}

type applyPoolRequest struct {
	Metadata schema.Metadata   `json:"metadata"`
	Spec     poolv1.PoolSpecV1 `json:"spec"`
}

func (a *Adapter) applyPool(w http.ResponseWriter, r *http.Request) {
	var req applyPoolRequest
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

	pool, err := a.cp.ApplyPoolV1(r.Context(), req.Metadata, req.Spec)
	if err != nil {
		writeCPError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, pool)
}

func (a *Adapter) deletePool(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := a.cp.DeletePool(r.Context(), name); err != nil {
		writeCPError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
