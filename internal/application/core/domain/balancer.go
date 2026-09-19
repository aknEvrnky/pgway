package domain

import "time"

type BalancerType string

const (
	BalancerTypeRoundRobin BalancerType = "round-robin"
	BalancerTypeWeighted   BalancerType = "weighted"
	BalancerTypeLeastBytes BalancerType = "least-bytes"

	// DefaultLeastBytesResetInterval is the single default for least-bytes
	// counter windows (schema resolve + algorithm fallback).
	DefaultLeastBytesResetInterval = time.Minute
)

func (b BalancerType) IsValid() bool {
	switch b {
	case BalancerTypeRoundRobin, BalancerTypeWeighted, BalancerTypeLeastBytes:
		return true
	}
	return false
}

type LoadBalancer struct {
	Timestamps
	Id            string        `json:"id"`
	Title         string        `json:"title"`
	Type          BalancerType  `json:"type"`
	PoolId        string        `json:"pool_id"`
	ResetInterval time.Duration `json:"reset_interval,omitempty"` // least-bytes only; 0 → DefaultLeastBytesResetInterval
}

type BalancerResult struct {
	ProxyId string
	Bytes   int64
}
