package domain

type Flow struct {
	Id         string `json:"string"`
	RouterId   string `json:"router,omitempty"`
	BalancerId string `json:"balancer"`
}
