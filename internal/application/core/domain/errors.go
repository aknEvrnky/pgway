package domain

import "errors"

var (
	ErrNoProxy         = errors.New("no proxy")
	ErrNoPool          = errors.New("no pool")
	ErrNoMatchingRule  = errors.New("no matching rule found")
	ErrAgentExists     = errors.New("agent already exists")
	ErrTokenMustExpire = errors.New("token must have expiration date")
)
