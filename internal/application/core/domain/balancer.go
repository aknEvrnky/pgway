package domain

type BalancerType string

const (
	BalancerTypeRoundRobin BalancerType = "round-robin"
	BalancerTypeWeighted   BalancerType = "weighted"
	BalancerTypeLeastBytes BalancerType = "least-bytes"
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
	Id     string       `json:"id"`
	Title  string       `json:"title"`
	Type   BalancerType `json:"type"`
	PoolId string       `json:"pool_id"`
}

type BalancerResult struct {
	ProxyId string
	Bytes   int64
}
