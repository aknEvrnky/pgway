package domain

import "fmt"

type Protocol int

const (
	Http = iota
)

func (p Protocol) String() string {
	return [...]string{"http"}[p]
}

type Entrypoint struct {
	Id       string `json:"id"`
	Title    string `json:"title"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Flow     string `json:"flow"`
}

func (e *Entrypoint) ListenAddr() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}
