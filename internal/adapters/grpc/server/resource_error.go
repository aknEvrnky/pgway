package server

import (
	"errors"

	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mapResourceError maps control-plane resource errors to gRPC status codes.
func mapResourceError(op string, err error) error {
	var inUse *controlplane.ResourceInUseError
	if errors.As(err, &inUse) {
		return status.Errorf(codes.FailedPrecondition, "%s: %v", op, err)
	}
	return status.Errorf(codes.Internal, "%s: %v", op, err)
}
