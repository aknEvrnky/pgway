package domain

type ProxyFilter struct {
	Search   string
	Protocol string
	Labels   map[string]string
}

type PoolFilter struct {
	Search string
	Type   string
}

type BalancerFilter struct {
	Search string
	Type   string
	PoolId string
}

type RouterFilter struct {
	Search string
}

type EntrypointFilter struct {
	Search   string
	Protocol string
	Host     string
}

type FlowFilter struct {
	Search     string
	RouterId   string
	BalancerId string
}
