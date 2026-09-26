package rest

import (
	"errors"
	"net/http"
	"strings"

	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

// writeCPError maps control-plane errors to HTTP status codes for the dashboard.
func writeCPError(w http.ResponseWriter, err error) {
	var inUse *controlplane.ResourceInUseError
	if errors.As(err, &inUse) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	var missing *controlplane.ResourceMissingRefError
	if errors.As(err, &missing) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, domain.ErrNotFound) || strings.Contains(err.Error(), "not found") {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}
