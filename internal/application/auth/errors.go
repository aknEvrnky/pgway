package auth

import "errors"

var (
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrInvalidToken             = errors.New("invalid or expired token")
	ErrInvalidBootstrapToken    = errors.New("invalid bootstrap token")
	ErrAlreadyInitialized       = errors.New("already initialized: admin user exists")
	ErrUserExists               = errors.New("user already exists")
	ErrLastAdmin                = errors.New("cannot delete the last admin user")
	ErrWeakPassword             = errors.New("password must be at least 8 characters")
	ErrInvalidRegistrationToken = errors.New("invalid registration token")
)
