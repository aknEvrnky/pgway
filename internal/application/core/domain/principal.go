package domain

type PrincipalKind string

const (
	PrincipalKindUser  PrincipalKind = "user"
	PrincipalKindAgent PrincipalKind = "agent"
)

type Principal struct {
	User    *User
	AgentId string
}

func (p *Principal) Kind() PrincipalKind {
	if p.User != nil {
		return PrincipalKindUser
	}

	return PrincipalKindAgent
}
