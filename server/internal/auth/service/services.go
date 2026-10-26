// Package service provides specialized domain services for authentication, sessions, profiles, and administration.
package service

import (
	"server/internal/auth/config"
	"server/internal/auth/model"
	"server/internal/auth/repository"
	"server/internal/auth/token"
)

// Services encapsulates all individual auth domain services.
type Services struct {
	Signup   *SignupService
	Session  *SessionService
	Password *PasswordService
	Profile  *ProfileService
	Admin    *UserAdminService
}

// NewServices instantiates and wires all domain services.
func NewServices(
	repo repository.AuthRepository,
	cfg config.AuthConfig,
	jwtIssuer *token.JWTIssuer,
	mailer model.Mailer,
) *Services {
	return &Services{
		Signup:   NewSignupService(repo, cfg, jwtIssuer, mailer),
		Session:  NewSessionService(repo, cfg, jwtIssuer),
		Password: NewPasswordService(repo, cfg, mailer),
		Profile:  NewProfileService(repo, cfg),
		Admin:    NewUserAdminService(repo),
	}
}
