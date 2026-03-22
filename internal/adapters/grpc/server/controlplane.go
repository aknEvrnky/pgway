package server

import (
	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/aknEvrnky/pgway/internal/schema"
)

type ControlPlaneServer struct {
	controlplanev1.UnimplementedProxyServiceServer
	controlplanev1.UnimplementedPoolServiceServer
	controlplanev1.UnimplementedRouterServiceServer
	controlplanev1.UnimplementedBalancerServiceServer
	controlplanev1.UnimplementedEntrypointServiceServer
	controlplanev1.UnimplementedFlowServiceServer

	cp ports.ControlPlane
}

func NewControlPlaneServer(cp ports.ControlPlane) *ControlPlaneServer {
	return &ControlPlaneServer{
		cp: cp,
	}
}

func metaFromProto(pb *controlplanev1.Metadata) schema.Metadata {
	if pb == nil {
		return schema.Metadata{}
	}

	return schema.Metadata{
		Name:   pb.Name,
		Labels: pb.Labels,
	}
}
