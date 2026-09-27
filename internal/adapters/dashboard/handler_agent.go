package dashboard

import (
	"net/http"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

type agentItemResponse struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"`
	Hostname      string            `json:"hostname,omitempty"`
	Version       string            `json:"version,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	LastHeartbeat *time.Time        `json:"last_heartbeat,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type listAgentsResponse struct {
	Items      []agentItemResponse `json:"items"`
	NextCursor string              `json:"next_cursor,omitempty"`
	TotalCount int                 `json:"total_count"`
}

func (a *Adapter) listAgents(w http.ResponseWriter, r *http.Request) {
	if a.agents == nil {
		writeError(w, http.StatusNotImplemented, "agents not available")
		return
	}

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

	result, err := a.agents.ListAgents(r.Context(), domain.ListParams{
		PageSize: pageSize,
		Cursor:   cursor,
	}, domain.AgentFilter{
		Search: q.Get("search"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	now := time.Now()
	items := make([]agentItemResponse, 0, len(result.Items))
	for _, ag := range result.Items {
		items = append(items, agentItemResponse{
			ID:            ag.Id,
			Status:        string(ag.Status(now, a.heartbeatThreshold)),
			Hostname:      ag.Hostname,
			Version:       ag.Version,
			Labels:        ag.Labels,
			LastHeartbeat: ag.LastHeartbeat,
			CreatedAt:     ag.CreatedAt,
			UpdatedAt:     ag.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, listAgentsResponse{
		Items:      items,
		NextCursor: encodePageToken(result.NextCursor),
		TotalCount: result.TotalCount,
	})
}
