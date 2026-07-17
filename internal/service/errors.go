package service

import "errors"

var (
	// ErrNotFound indicates a requested entity does not exist.
	ErrNotFound = errors.New("service: not found")
	// ErrInvalidInput indicates caller input failed validation.
	ErrInvalidInput = errors.New("service: invalid input")
	// ErrInsufficientPoints indicates the user cannot afford the operation.
	ErrInsufficientPoints = errors.New("service: insufficient points")
	// ErrBanned indicates the user is banned.
	ErrBanned = errors.New("service: user is banned")
	// ErrForbidden indicates the actor lacks privileges.
	ErrForbidden = errors.New("service: forbidden")
	// ErrUnauthorized indicates the actor is not allowed to use the bot.
	ErrUnauthorized = errors.New("service: unauthorized")
	// ErrRateLimited indicates the actor exceeded the allowed request rate.
	ErrRateLimited = errors.New("service: rate limited")
	// ErrNotSupported indicates the operation is unavailable with current ports.
	ErrNotSupported = errors.New("service: not supported")
)
