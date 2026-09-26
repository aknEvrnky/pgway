package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	routerv1 "github.com/aknEvrnky/pgway/internal/schema/router/v1"
)

type listRoutersResponse struct {
	Items      []*domain.Router `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
	TotalCount int              `json:"total_count"`
}

func (a *Adapter) listRouters(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

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

	filter := domain.RouterFilter{
		Search:           q.Get("search"),
		TargetBalancerId: q.Get("target"),
	}
	if raw := q.Get("has_catch_all"); raw != "" {
		switch raw {
		case "true", "1":
			v := true
			filter.HasCatchAll = &v
		case "false", "0":
			v := false
			filter.HasCatchAll = &v
		default:
			writeError(w, http.StatusBadRequest, `has_catch_all must be "true" or "false"`)
			return
		}
	}

	result, err := a.cp.ListRouters(r.Context(), domain.ListParams{
		PageSize: pageSize,
		Cursor:   cursor,
	}, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, listRoutersResponse{
		Items:      result.Items,
		NextCursor: encodePageToken(result.NextCursor),
		TotalCount: result.TotalCount,
	})
}

func (a *Adapter) getRouter(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	router, err := a.cp.GetRouter(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, router)
}

type applyRouterRequest struct {
	Metadata schema.Metadata       `json:"metadata"`
	Spec     routerv1.RouterSpecV1 `json:"spec"`
}

func (a *Adapter) applyRouter(w http.ResponseWriter, r *http.Request) {
	var req applyRouterRequest
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

	router, err := a.cp.ApplyRouterV1(r.Context(), req.Metadata, req.Spec)
	if err != nil {
		writeCPError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, router)
}

func (a *Adapter) deleteRouter(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := a.cp.DeleteRouter(r.Context(), name); err != nil {
		writeCPError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
