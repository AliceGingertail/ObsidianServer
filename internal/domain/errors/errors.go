package errors

import "errors"

var (
	// Auth errors
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	
	// Peer errors
	ErrPeerNotFound       = errors.New("peer not found")
	ErrMaxPeersReached    = errors.New("maximum peers limit reached")
	ErrInvalidProtocol    = errors.New("invalid VPN protocol")
	ErrProviderNotFound   = errors.New("VPN provider not found")
	
	// Generic errors
	ErrNotFound           = errors.New("resource not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrBadRequest         = errors.New("bad request")
	ErrInternalServer     = errors.New("internal server error")
)
