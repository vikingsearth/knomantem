// Package domain contains the core entities, value objects, repository
// interfaces, and domain errors for Knomantem.
package domain

import "errors"

// Sentinel domain errors. Handlers map these to HTTP status codes.
var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("not found")

	// ErrConflict is returned when an operation would violate a uniqueness constraint.
	ErrConflict = errors.New("conflict")

	// ErrValidation is returned when input fails domain-level validation rules.
	ErrValidation = errors.New("validation error")

	// ErrUnauthorized is returned when authentication is required but not provided.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when the authenticated user lacks permission.
	ErrForbidden = errors.New("forbidden")
)
