package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
)

// balancerResponse mirrors domain.LoadBalancer for JSON, with reset_interval as a
// Go duration string (same wire shape as gRPC) instead of nanoseconds.
type balancerResponse struct {
	Id            string    `json:"id"`
	Title         string    `json:"title"`
	Type          string    `json:"type"`
	PoolId        string    `json:"pool_id"`
	ResetInterval string    `json:"reset_interval,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func mapBalancer(lb *domain.LoadBalancer) balancerResponse {
	if lb == nil {
		return balancerResponse{}
	}
	out := balancerResponse{
		Id:        lb.Id,
		Title:     lb.Title,
		Type:      string(lb.Type),
		PoolId:    lb.PoolId,
		CreatedAt: lb.CreatedAt,
		UpdatedAt: lb.UpdatedAt,
	}
	if lb.Type == domain.BalancerTypeLeastBytes && lb.ResetInterval > 0 {
		out.ResetInterval = compactDuration(lb.ResetInterval)
	}
	return out
}

// compactDuration prefers "1m" over Go's default "1m0s".
func compactDuration(d time.Duration) string {
	s := d.String()
	if strings.HasSuffix(s, "m0s") {
		return strings.TrimSuffix(s, "0s")
	}
	return s
}

type listBalancersResponse struct {
	Items      []balancerResponse `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
	TotalCount int                `json:"total_count"`
}

func (a *Adapter) listBalancers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	lbType := q.Get("type")
	if lbType != "" {
		bt := domain.BalancerType(lbType)
		if !bt.IsValid() {
			writeError(w, http.StatusBadRequest, `type must be "round-robin", "weighted", or "least-bytes"`)
			return
		}
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

	result, err := a.cp.ListBalancers(r.Context(), domain.ListParams{
		PageSize: pageSize,
		Cursor:   cursor,
	}, domain.BalancerFilter{
		Search: q.Get("search"),
		Type:   lbType,
		PoolId: q.Get("pool_id"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]balancerResponse, 0, len(result.Items))
	for _, lb := range result.Items {
		items = append(items, mapBalancer(lb))
	}

	writeJSON(w, http.StatusOK, listBalancersResponse{
		Items:      items,
		NextCursor: encodePageToken(result.NextCursor),
		TotalCount: result.TotalCount,
	})
}

func (a *Adapter) getBalancer(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	lb, err := a.cp.GetBalancer(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, mapBalancer(lb))
}

type applyBalancerRequest struct {
	Metadata schema.Metadata           `json:"metadata"`
	Spec     balancerv1.BalancerSpecV1 `json:"spec"`
}

func (a *Adapter) applyBalancer(w http.ResponseWriter, r *http.Request) {
	var req applyBalancerRequest
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

	lb, err := a.cp.ApplyBalancerV1(r.Context(), req.Metadata, req.Spec)
	if err != nil {
		writeCPError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapBalancer(lb))
}

func (a *Adapter) deleteBalancer(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := a.cp.DeleteBalancer(r.Context(), name); err != nil {
		writeCPError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
