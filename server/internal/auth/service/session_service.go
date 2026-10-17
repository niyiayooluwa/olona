package service

import (
	"context"
	"errors"
	"time"

	"server/internal/auth/config"
	autherr "server/internal/auth/errors"
	"server/internal/auth/model"
	"server/internal/auth/repository"
	"server/internal/auth/service/helpers"
	"server/internal/auth/utils"
	db "server/internal/db/generated"
)

// SessionService handles authentication, session renewal, token revocation, and OAuth logins.
type SessionService struct {
	repo      repository.AuthRepository
	config    config.AuthConfig
	jwtIssuer *utils.JWTIssuer
}

// NewSessionService constructs a new SessionService.
func NewSessionService(
	repo repository.AuthRepository,
	cfg config.AuthConfig,
	jwtIssuer *utils.JWTIssuer,
) *SessionService {
	return &SessionService{
		repo:      repo,
		config:    cfg,
		jwtIssuer: jwtIssuer,
	}
}

// Login authenticates a user using their email and plaintext password.
//
// Security guarantees:
//   - Verifies PasswordConfig.EnablePasswordLogin is true.
//   - Side-Channel Mitigation (AUTH-17): If the user does not exist in the database,
//     the service executes a dummy bcrypt comparison (DummyPasswordCompare) against a precomputed
//     hash so that the server's response time is identical for valid and invalid emails.
//   - Rejects OAuth-only accounts that do not have a password set with ErrOAuthAccount.
//   - Enforces suspension (ErrAccountSuspended) and deactivation (ErrAccountDeactivated).
//   - Verification Gate: If VerificationConfig.Required is true, blocks unverified accounts
//     with ErrEmailNotVerified.
//   - Returns an active TokenPair (access token + refresh token) on successful authentication.
//
// Returns:
// - (*model.TokenPair, nil): Credentials valid and account active.
// - (nil, ErrPasswordLoginDisabled): Password login disabled in system configuration.
// - (nil, ErrInvalidCredentials): Wrong password or non-existent email address.
// - (nil, ErrOAuthAccount): Account created via OAuth; must sign in via provider.
// - (nil, ErrAccountSuspended): Account is suspended.
// - (nil, ErrAccountDeactivated): Account is deactivated.
// - (nil, ErrEmailNotVerified): Account email has not been verified yet.
func (service *SessionService) Login(ctx context.Context, email, password string) (*model.TokenPair, error) {
	// 1. Enforce password login feature flag
	if !service.config.Password.EnablePasswordLogin {
		return nil, autherr.ErrPasswordLoginDisabled
	}

	// 2. Normalize submitted email address
	normalizedEmail := helpers.NormalizeEmail(email)

	// 3. Query repository for user account
	user, err := service.repo.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Side-channel defense: perform dummy bcrypt check to equalize timing profile
			utils.DummyPasswordCompare(password)
			return nil, autherr.ErrInvalidCredentials
		}
		return nil, err
	}

	// 4. Reject accounts created via third-party OAuth providers that lack a password credential
	if !user.PasswordHash.Valid {
		utils.DummyPasswordCompare(password)
		return nil, autherr.ErrOAuthAccount
	}

	// 5. Verify submitted password against stored bcrypt hash using constant-time check
	if !utils.CheckPassword(user.PasswordHash.String, password) {
		return nil, autherr.ErrInvalidCredentials
	}

	// 6. Enforce account status restrictions
	if user.IsSuspended {
		return nil, autherr.ErrAccountSuspended
	}
	if user.IsDeactivated {
		return nil, autherr.ErrAccountDeactivated
	}

	// 7. Enforce email verification requirement
	if service.config.Verification.Required && user.EmailVerifiedAt == nil {
		return nil, autherr.ErrEmailNotVerified
	}

	// 8. Issue active session token pair (access token + refresh token)
	return helpers.IssueTokenPair(ctx, service.repo, service.jwtIssuer, service.config.Session, user.ID)
}

// Refresh validates an existing refresh token and issues a fresh session pair.
//
// Security guarantees:
//   - Token Expiration: Verifies current time does not exceed the token's expires_at.
//   - Replay / Theft Detection (AUTH-15): If a revoked token (revoked_at IS NOT NULL) is presented,
//     the service treats this as a session hijacking event and IMMEDIATELY revokes ALL active
//     refresh tokens for that user across all devices (RevokeAllUserRefreshTokens), returning
//     ErrRefreshTokenReused.
//   - Enforces user status: rejects requests if account was suspended or deactivated since token issuance.
//   - Rotation (AUTH-14): If SessionConfig.RotateRefreshToken is enabled, marks the presented refresh
//     token as revoked and issues a brand-new token pair. If disabled, re-issues the access token only.
//
// Returns:
// - (*model.TokenPair, nil): Session refreshed successfully.
// - (nil, ErrRefreshTokenInvalid): Token is unknown, corrupted, expired, or belongs to invalid user.
// - (nil, ErrRefreshTokenReused): Revoked token presented; all user sessions terminated.
// - (nil, ErrAccountSuspended / ErrAccountDeactivated): Account is locked.
func (service *SessionService) Refresh(ctx context.Context, rawRefreshToken string) (*model.TokenPair, error) {
	// 1. Compute SHA-256 hash of presented raw refresh token
	tokenHash := utils.HashToken(rawRefreshToken)

	// 2. Fetch token record from repository
	tokenRecord, err := service.repo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, autherr.ErrRefreshTokenInvalid
		}
		return nil, err
	}

	// 3. Verify token has not expired
	if time.Now().After(tokenRecord.ExpiresAt) {
		return nil, autherr.ErrRefreshTokenInvalid
	}

	// 4. Theft detection: presenting an already-revoked token indicates token replay; nuke all sessions
	if tokenRecord.RevokedAt != nil {
		if err := service.repo.RevokeAllUserRefreshTokens(ctx, tokenRecord.UserID); err != nil {
			return nil, err
		}
		return nil, autherr.ErrRefreshTokenReused
	}

	// 5. Validate current user status
	user, err := service.repo.GetUserByID(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, autherr.ErrRefreshTokenInvalid
	}
	if user.IsSuspended {
		return nil, autherr.ErrAccountSuspended
	}
	if user.IsDeactivated {
		return nil, autherr.ErrAccountDeactivated
	}

	// 6. Handle refresh token rotation if enabled
	if service.config.Session.RotateRefreshToken {
		// Revoke the presented refresh token
		if err := service.repo.RevokeRefreshToken(ctx, tokenRecord.ID); err != nil {
			return nil, err
		}
		// Issue a brand-new access token + refresh token pair
		return helpers.IssueTokenPair(ctx, service.repo, service.jwtIssuer, service.config.Session, user.ID)
	}

	// 7. Non-rotating mode: issue new short-lived access token, reuse existing refresh token
	accessToken, err := service.jwtIssuer.Issue(user.ID, service.config.Session.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
	}, nil
}

// Logout terminates a user's session by revoking the specified refresh token.
//
// Behavior:
// - Hashes the raw refresh token with SHA-256.
// - Locates the token record in the database.
// - Marks revoked_at = now().
// - Idempotent: returns nil without error if the token was already revoked or not found.
func (service *SessionService) Logout(ctx context.Context, rawRefreshToken string) error {
	// 1. Hash raw refresh token with SHA-256
	tokenHash := utils.HashToken(rawRefreshToken)

	// 2. Look up token record
	tokenRecord, err := service.repo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil // Idempotent logout: unknown token returns nil
		}
		return err
	}

	// 3. Revoke the token record in database
	return service.repo.RevokeRefreshToken(ctx, tokenRecord.ID)
}

// OAuthLogin handles authentication via an external identity provider (e.g. Google).
//
// Lifecycle & Provisioning behavior:
//  1. Verifies that OAuth is globally enabled and the requested provider is registered.
//  2. Validates the incoming ID token with the provider, obtaining a verified Identity.
//  3. Existing Connection: If an oauth_connections row already matches (provider, provider_user_id),
//     the linked user is validated (suspended/deactivated checks) and active session tokens are issued.
//  4. Account Linking: If no connection exists but a user account with the verified email is found,
//     the OAuth connection is linked to that existing account and active session tokens are issued.
//  5. Just-In-Time Provisioning: If no user exists, verifies registration policy (open mode + OAuth enabled),
//     creates a new user with verified email (inbox ownership proven by OAuth provider), creates a profile
//     with available identity data (name, avatar), links the provider connection, and issues session tokens.
//
// Returns:
// - (*model.TokenPair, nil): Authenticated successfully.
// - (nil, ErrOAuthProviderNotSupported): Provider is unknown or disabled.
// - (nil, ErrOAuthTokenInvalid): Token signature or claims verification failed.
// - (nil, ErrRegistrationDisabled): New user provisioning blocked by closed registration mode.
// - (nil, ErrAccountSuspended / ErrAccountDeactivated): Account is locked.
func (service *SessionService) OAuthLogin(ctx context.Context, providerName, idToken string) (*model.TokenPair, error) {
	// 1. Enforce global OAuth configuration and provider availability
	if !service.config.OAuth.Enabled {
		return nil, autherr.ErrOAuthProviderNotSupported
	}

	provider, exists := service.config.OAuth.Providers[providerName]
	if !exists || provider == nil {
		return nil, autherr.ErrOAuthProviderNotSupported
	}

	// 2. Verify third-party ID token with the provider's validator
	identity, err := provider.VerifyToken(ctx, idToken)
	if err != nil {
		return nil, autherr.ErrOAuthTokenInvalid
	}

	normalizedEmail := helpers.NormalizeEmail(identity.Email)

	// 3. Check if OAuth connection already exists for this provider and subject ID
	connection, err := service.repo.GetOAuthConnection(ctx, providerName, identity.ProviderID)
	if err == nil {
		// Existing OAuth connection found; fetch linked user account
		user, err := service.repo.GetUserByID(ctx, connection.UserID)
		if err != nil {
			return nil, err
		}
		// Enforce user account status
		if user.IsSuspended {
			return nil, autherr.ErrAccountSuspended
		}
		if user.IsDeactivated {
			return nil, autherr.ErrAccountDeactivated
		}
		// Issue active session token pair
		return helpers.IssueTokenPair(ctx, service.repo, service.jwtIssuer, service.config.Session, user.ID)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	// 4. Check if a user with this verified email already exists in the system
	existingUser, err := service.repo.GetUserByEmail(ctx, normalizedEmail)
	if err == nil {
		// Link the new OAuth provider connection to the existing user account
		_, err = service.repo.CreateOAuthConnection(ctx, existingUser.ID, providerName, identity.ProviderID)
		if err != nil {
			return nil, err
		}
		// Enforce user account status
		if existingUser.IsSuspended {
			return nil, autherr.ErrAccountSuspended
		}
		if existingUser.IsDeactivated {
			return nil, autherr.ErrAccountDeactivated
		}
		// Issue active session token pair
		return helpers.IssueTokenPair(ctx, service.repo, service.jwtIssuer, service.config.Session, existingUser.ID)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	// 5. New account provisioning: enforce registration policy
	if !service.config.Registration.EnableOAuthRegistration || service.config.Registration.Mode != config.RegistrationModeOpen {
		return nil, autherr.ErrRegistrationDisabled
	}

	// 6-8. Create user, profile, and OAuth connection atomically
	var newUser db.User
	err = service.repo.WithTx(ctx, func(txRepo repository.AuthRepository) error {
		var txErr error
		newUser, txErr = txRepo.CreateUser(ctx, normalizedEmail, nil)
		if txErr != nil {
			return txErr
		}
		if txErr = txRepo.MarkEmailVerified(ctx, newUser.ID); txErr != nil {
			return txErr
		}

		var firstName, lastName, avatarURL *string
		if service.config.Profile.EnableName {
			if identity.FirstName != "" {
				firstName = &identity.FirstName
			}
			if identity.LastName != "" {
				lastName = &identity.LastName
			}
		}
		if identity.AvatarURL != "" {
			avatarURL = &identity.AvatarURL
		}

		if _, txErr = txRepo.CreateProfile(ctx, newUser.ID, firstName, lastName, nil, nil, nil, avatarURL); txErr != nil {
			return txErr
		}

		_, txErr = txRepo.CreateOAuthConnection(ctx, newUser.ID, providerName, identity.ProviderID)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	// 9. Issue active session token pair (access + refresh)
	return helpers.IssueTokenPair(ctx, service.repo, service.jwtIssuer, service.config.Session, newUser.ID)
}
