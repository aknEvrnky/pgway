package domain

type Flow struct {
	Timestamps
	Id         string `json:"id"`
	RouterId   string `json:"router_id,omitempty"`
	BalancerId string `json:"balancer_id,omitempty"`
}
