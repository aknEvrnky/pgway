package server

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *ControlPlaneServer) ApplyProxyV1(ctx context.Context, req *controlplanev1.ApplyProxyV1Request) (*controlplanev1.ApplyProxyV1Response, error) {
	if req.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "spec is required")
	}

	meta := metaFromProto(req.Metadata)
	spec := proxySpecFromProto(req.Spec)

	proxy, err := s.cp.ApplyProxyV1(ctx, meta, spec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "apply proxy: %v", err)
	}

	return &controlplanev1.ApplyProxyV1Response{
		Proxy: proxyToProto(proxy),
	}, nil
}

func (s *ControlPlaneServer) GetProxy(ctx context.Context, req *controlplanev1.GetProxyRequest) (*controlplanev1.GetProxyResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	proxy, err := s.cp.GetProxy(ctx, req.Name)

	if err != nil {
		return nil, status.Errorf(codes.NotFound, "proxy: %v", err)
	}

	return &controlplanev1.GetProxyResponse{
		Proxy: proxyToProto(proxy),
	}, nil
}

func (s *ControlPlaneServer) ListProxies(ctx context.Context, req *controlplanev1.ListProxiesRequest) (*controlplanev1.ListProxiesResponse, error) {
	cursor, err := decodeCursor(req.GetPageToken())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid page_token")
	}

	params := domain.ListParams{
		PageSize: int(req.GetPageSize()),
		Cursor:   cursor,
	}

	filter := domain.ProxyFilter{
		Search:   req.GetSearch(),
		Protocol: req.GetProtocol(),
		Labels:   req.GetLabels(),
	}

	result, err := s.cp.ListProxies(ctx, params, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list proxies: %v", err)
	}

	proxies := make([]*controlplanev1.Proxy, 0, len(result.Items))
	for _, p := range result.Items {
		proxies = append(proxies, proxyToProto(p))
	}

	return &controlplanev1.ListProxiesResponse{
		Proxies:       proxies,
		NextPageToken: encodeCursor(result.NextCursor),
		TotalCount:    int32(result.TotalCount),
	}, nil
}

func (s *ControlPlaneServer) DeleteProxy(ctx context.Context, req *controlplanev1.DeleteProxyRequest) (*controlplanev1.DeleteProxyResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	err := s.cp.DeleteProxy(ctx, req.Name)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete proxy: %v", err)
	}

	return &controlplanev1.DeleteProxyResponse{}, nil
}

func (s *ControlPlaneServer) GetProxiesByIds(ctx context.Context, req *controlplanev1.GetProxiesByIdsRequest) (*controlplanev1.GetProxiesByIdsResponse, error) {
	if len(req.Ids) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ids are required")
	}

	proxies, err := s.cp.GetProxiesByIds(ctx, req.Ids)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "get proxies by ids: %q %v", req.Ids, err)
	}

	var proxyPbs = make([]*controlplanev1.Proxy, 0, len(proxies))

	for _, p := range proxies {
		proxyPbs = append(proxyPbs, proxyToProto(p))
	}

	return &controlplanev1.GetProxiesByIdsResponse{
		Proxies: proxyPbs,
	}, nil
}

func (s *ControlPlaneServer) FindProxiesByLabels(ctx context.Context, req *controlplanev1.FindProxiesByLabelsRequest) (*controlplanev1.FindProxiesByLabelsResponse, error) {
	if len(req.Labels) == 0 {
		return nil, status.Error(codes.InvalidArgument, "labels are required")
	}

	proxies, err := s.cp.FindProxiesByLabels(ctx, req.Labels)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "find proxies by labels: %q %v", req.Labels, err)
	}

	var proxyPbs = make([]*controlplanev1.Proxy, 0, len(proxies))

	for _, p := range proxies {
		proxyPbs = append(proxyPbs, proxyToProto(p))
	}

	return &controlplanev1.FindProxiesByLabelsResponse{
		Proxies: proxyPbs,
	}, nil
}

func proxySpecFromProto(pb *controlplanev1.ProxySpecV1) proxyv1.ProxySpecV1 {
	if pb == nil {
		return proxyv1.ProxySpecV1{}
	}

	return proxyv1.ProxySpecV1{
		URL:      pb.Url,
		Protocol: pb.Protocol,
		Host:     pb.Host,
		Port:     uint16(pb.Port),
		Auth: func() *proxyv1.AuthSpec {
			if pb.Auth == nil {
				return nil
			}

			return &proxyv1.AuthSpec{
				User: pb.Auth.User,
				Pass: pb.Auth.User,
			}
		}(),
	}
}

func proxyToProto(proxy *domain.Proxy) *controlplanev1.Proxy {
	if proxy == nil {
		return nil
	}

	return &controlplanev1.Proxy{
		Id:       proxy.Id,
		Protocol: string(proxy.Protocol),
		Host:     proxy.Host,
		Port:     uint32(proxy.Port),
		Auth: func() *controlplanev1.AuthSpec {
			if proxy.Auth == nil {
				return nil
			}

			return &controlplanev1.AuthSpec{
				User: proxy.Auth.User,
				Pass: proxy.Auth.Pass,
			}
		}(),
		Labels:    proxy.Labels,
		CreatedAt: timestamppb.New(proxy.CreatedAt),
		UpdatedAt: timestamppb.New(proxy.UpdatedAt),
	}
}
