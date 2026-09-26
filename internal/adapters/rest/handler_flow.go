package rest

import (
	"encoding/json"
	"net/http"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
)

type listFlowsResponse struct {
	Items      []*domain.Flow `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
	TotalCount int            `json:"total_count"`
}

func (a *Adapter) listFlows(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	mode := q.Get("mode")
	if mode != "" && mode != "router" && mode != "direct" {
		writeError(w, http.StatusBadRequest, `mode must be "router" or "direct"`)
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

	result, err := a.cp.ListFlows(r.Context(), domain.ListParams{
		PageSize: pageSize,
		Cursor:   cursor,
	}, domain.FlowFilter{
		Search:     q.Get("search"),
		RouterId:   q.Get("router_id"),
		BalancerId: q.Get("balancer_id"),
		Mode:       mode,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, listFlowsResponse{
		Items:      result.Items,
		NextCursor: encodePageToken(result.NextCursor),
		TotalCount: result.TotalCount,
	})
}

func (a *Adapter) getFlow(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	flow, err := a.cp.GetFlow(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, flow)
}

type applyFlowRequest struct {
	Metadata schema.Metadata   `json:"metadata"`
	Spec     flowv1.FlowSpecV1 `json:"spec"`
}

func (a *Adapter) applyFlow(w http.ResponseWriter, r *http.Request) {
	var req applyFlowRequest
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

	flow, err := a.cp.ApplyFlowV1(r.Context(), req.Metadata, req.Spec)
	if err != nil {
		writeCPError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, flow)
}

func (a *Adapter) deleteFlow(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := a.cp.DeleteFlow(r.Context(), name); err != nil {
		writeCPError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
