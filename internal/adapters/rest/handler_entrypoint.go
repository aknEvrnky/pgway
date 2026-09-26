package rest

import (
	"encoding/json"
	"net/http"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	entrypointv1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
)

type listEntrypointsResponse struct {
	Items      []*domain.Entrypoint `json:"items"`
	NextCursor string               `json:"next_cursor,omitempty"`
	TotalCount int                  `json:"total_count"`
}

func (a *Adapter) listEntrypoints(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

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

	result, err := a.cp.ListEntrypoints(r.Context(), domain.ListParams{
		PageSize: pageSize,
		Cursor:   cursor,
	}, domain.EntrypointFilter{
		Search:   q.Get("search"),
		Protocol: q.Get("protocol"),
		Host:     q.Get("host"),
		FlowId:   q.Get("flow_id"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, listEntrypointsResponse{
		Items:      result.Items,
		NextCursor: encodePageToken(result.NextCursor),
		TotalCount: result.TotalCount,
	})
}

func (a *Adapter) getEntrypoint(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	ep, err := a.cp.GetEntrypoint(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ep)
}

type applyEntrypointRequest struct {
	Metadata schema.Metadata               `json:"metadata"`
	Spec     entrypointv1.EntrypointSpecV1 `json:"spec"`
}

func (a *Adapter) applyEntrypoint(w http.ResponseWriter, r *http.Request) {
	var req applyEntrypointRequest
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

	ep, err := a.cp.ApplyEntrypointV1(r.Context(), req.Metadata, req.Spec)
	if err != nil {
		writeCPError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, ep)
}

func (a *Adapter) deleteEntrypoint(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := a.cp.DeleteEntrypoint(r.Context(), name); err != nil {
		writeCPError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
