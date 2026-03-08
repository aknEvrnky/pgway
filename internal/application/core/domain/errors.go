package domain

import "errors"

var (
	ErrNoProxy        = errors.New("no proxy")
	ErrNoMatchingRule = errors.New("no matching rule found")
)
