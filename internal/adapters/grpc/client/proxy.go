package client

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
)

func (c *Client) ApplyProxyV1(ctx context.Context, meta schema.Metadata, spec proxyv1.ProxySpecV1) (*domain.Proxy, error) {
	req := &controlplanev1.ApplyProxyV1Request{
		Metadata: metaToProto(meta),
		Spec:     proxySpecToProto(spec),
	}

	resp, err := c.proxy.ApplyProxyV1(ctx, req)
	if err != nil {
		return nil, err
	}

	return proxyFromProto(resp.Proxy), nil
}

func (c *Client) GetProxy(ctx context.Context, name string) (*domain.Proxy, error) {
	req := &controlplanev1.GetProxyRequest{Name: name}

	resp, err := c.proxy.GetProxy(ctx, req)
	if err != nil {
		return nil, err
	}

	return proxyFromProto(resp.Proxy), nil
}

func (c *Client) ListProxies(ctx context.Context) ([]*domain.Proxy, error) {
	req := &controlplanev1.ListProxiesRequest{}

	resp, err := c.proxy.ListProxies(ctx, req)
	if err != nil {
		return nil, err
	}

	var results = make([]*domain.Proxy, 0, len(resp.Proxies))
	for _, p := range resp.Proxies {
		results = append(results, proxyFromProto(p))
	}

	return results, nil
}

func (c *Client) DeleteProxy(ctx context.Context, name string) error {
	req := &controlplanev1.DeleteProxyRequest{Name: name}
	_, err := c.proxy.DeleteProxy(ctx, req)
	return err
}

func (c *Client) GetProxiesByIds(ctx context.Context, ids []string) ([]*domain.Proxy, error) {
	req := &controlplanev1.GetProxiesByIdsRequest{Ids: ids}

	resp, err := c.proxy.GetProxiesByIds(ctx, req)
	if err != nil {
		return nil, err
	}

	var results = make([]*domain.Proxy, 0, len(resp.Proxies))
	for _, p := range resp.Proxies {
		results = append(results, proxyFromProto(p))
	}

	return results, nil
}

func (c *Client) FindProxiesByLabels(ctx context.Context, labels map[string]string) ([]*domain.Proxy, error) {
	req := &controlplanev1.FindProxiesByLabelsRequest{Labels: labels}

	resp, err := c.proxy.FindProxiesByLabels(ctx, req)
	if err != nil {
		return nil, err
	}

	var results = make([]*domain.Proxy, 0, len(resp.Proxies))
	for _, p := range resp.Proxies {
		results = append(results, proxyFromProto(p))
	}

	return results, nil
}

func proxySpecToProto(spec proxyv1.ProxySpecV1) *controlplanev1.ProxySpecV1 {
	pb := &controlplanev1.ProxySpecV1{
		Url:      spec.URL,
		Protocol: spec.Protocol,
		Host:     spec.Host,
		Port:     uint32(spec.Port),
	}

	if spec.Auth != nil {
		pb.Auth = &controlplanev1.AuthSpec{
			User: spec.Auth.User,
			Pass: spec.Auth.Pass,
		}
	}

	return pb
}

func proxyFromProto(pb *controlplanev1.Proxy) *domain.Proxy {
	proxy := &domain.Proxy{
		Id:       pb.Id,
		Protocol: domain.Protocol(pb.Protocol),
		Host:     pb.Host,
		Port:     uint16(pb.Port),
		Labels:   pb.Labels,
	}

	if pb.Auth != nil {
		proxy.Auth = &domain.BasicAuth{
			User: pb.Auth.User,
			Pass: pb.Auth.Pass,
		}
	}

	return proxy
}
