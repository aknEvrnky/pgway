package domain

type Pool struct {
	Id      string   `json:"id"`
	Title   string   `json:"title"`
	Tags    []string `json:"tags"`
	Proxies []*Proxy `json:"proxies"`
}
