package client

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// KeepaliveConfig configures gRPC client keepalive pings.
// Interval <= 0 disables keepalive options entirely.
type KeepaliveConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

func keepaliveDialOptions(ka KeepaliveConfig) []grpc.DialOption {
	params, ok := clientKeepaliveParams(ka)
	if !ok {
		return nil
	}
	return []grpc.DialOption{grpc.WithKeepaliveParams(params)}
}

func clientKeepaliveParams(ka KeepaliveConfig) (keepalive.ClientParameters, bool) {
	if ka.Interval <= 0 {
		return keepalive.ClientParameters{}, false
	}
	return keepalive.ClientParameters{
		Time:                ka.Interval,
		Timeout:             ka.Timeout,
		PermitWithoutStream: true,
	}, true
}
