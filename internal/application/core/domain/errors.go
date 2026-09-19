package domain

import "errors"

var (
	ErrNoProxy         = errors.New("no proxy")
	ErrNoPool          = errors.New("no pool")
	ErrNoMatchingRule  = errors.New("no matching rule found")
	ErrNotFound        = errors.New("not found")
	ErrAgentExists     = errors.New("agent already exists")
	ErrUserExists      = errors.New("user already exists")
	ErrTokenMustExpire = errors.New("token must have expiration date")
)
