package server

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// Fixed secondary keepalive parameters (not configurable).
const (
	keepaliveMaxConnectionAge   = 30 * time.Minute
	keepaliveEnforcementMinTime = 30 * time.Second
)

// KeepaliveConfig configures gRPC server keepalive pings.
// Interval <= 0 disables keepalive options entirely.
type KeepaliveConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

func keepaliveServerOptions(ka KeepaliveConfig) []grpc.ServerOption {
	params, policy, ok := serverKeepaliveParams(ka)
	if !ok {
		return nil
	}
	return []grpc.ServerOption{
		grpc.KeepaliveParams(params),
		grpc.KeepaliveEnforcementPolicy(policy),
	}
}

func serverKeepaliveParams(ka KeepaliveConfig) (keepalive.ServerParameters, keepalive.EnforcementPolicy, bool) {
	if ka.Interval <= 0 {
		return keepalive.ServerParameters{}, keepalive.EnforcementPolicy{}, false
	}
	return keepalive.ServerParameters{
			Time:             ka.Interval,
			Timeout:          ka.Timeout,
			MaxConnectionAge: keepaliveMaxConnectionAge,
		},
		keepalive.EnforcementPolicy{
			MinTime: keepaliveEnforcementMinTime,
		},
		true
}
