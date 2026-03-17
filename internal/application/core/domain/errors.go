package domain

import "errors"

var (
	ErrNoProxy        = errors.New("no proxy")
	ErrNoPool         = errors.New("no pool")
	ErrNoMatchingRule = errors.New("no matching rule found")
)
