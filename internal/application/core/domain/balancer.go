package domain

type LoadBalancer struct {
	Id    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
	Pool  string `json:"pool"`
}
