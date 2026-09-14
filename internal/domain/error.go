package domain

import "errors"

var (
	// general
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("fobidden")

	// user / auth
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)
