package domain

type PrincipalKind string

const (
	PrincipalKindUser    PrincipalKind = "user"
	PrincipalKindAgent   PrincipalKind = "agent"
	PrincipalKindUnknown PrincipalKind = ""
)

type Principal struct {
	User  *User
	Agent *Agent
}

func (p *Principal) Kind() PrincipalKind {
	if p == nil {
		return PrincipalKindUnknown
	}
	if p.User != nil {
		return PrincipalKindUser
	}
	if p.Agent != nil {
		return PrincipalKindAgent
	}
	return PrincipalKindUnknown
}
