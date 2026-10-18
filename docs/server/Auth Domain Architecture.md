# Olona — Auth Domain Architecture & Reference

This document provides a comprehensive guide to the authentication and identity management service layer in `server/internal/auth`.

---

## 1. Architecture & Design Principles

The auth subsystem is designed as an **Off-The-Counter (OTC)** reusable module for Olona and future client applications. Rather than a monolithic God Object, it is decomposed into **6 specialized, domain-focused services** adhering to the Single Responsibility Principle:

```
┌────────────────────────────────────────────────────────────────────────┐
│                        HTTP Transport Layer                            │
│           (Gin / Fiber / Standard Library Handlers & DTOs)             │
│                                                                        │
│   AuthHandler        ProfileHandler      InviteHandler    AdminHandler │
│  (Signup, Session,     (Profile)          (Invitation)     (UserAdmin) │
│    Password)                                                           │
└────────┬─────────────────────┬──────────────────┬──────────────┬───────┘
         │                     │                  │              │
         ▼                     ▼                  ▼              ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        Focused Service Layer                           │
│  - SignupService     : Registration, OTP verification, invite accept   │
│  - SessionService    : Password login, token refresh, logout, OAuth    │
│  - PasswordService   : Forgot password, reset password, change password│
│  - ProfileService    : Profile CRUD, username uniqueness checks        │
│  - InvitationService : Issuing, inspecting, and revoking invitations   │
│  - UserAdminService  : Suspension, deactivation, user account deletion │
└────────┬────────────────────────────────────────┬──────────────────────┘
         │                                        │
         ▼                                        ▼
┌──────────────────────────────┐       ┌────────────────────────┐
│     Repository Layer         │       │    OAuth Providers     │
│       (repository)           │       │    (service/oauth)     │
│ - SQLC Query Execution       │       │ - Google ID Tokens     │
│ - SQLSTATE Error Translation │       │ - Extensible Provider  │
│ - Sentinel Error Wrapping    │       │   Interface            │
└────────┬─────────────────────┘       └────────────────────────┘
         │
         ▼
┌──────────────────────────────┐
│     PostgreSQL Database      │
└──────────────────────────────┘
```

### Core Invariants:
1. **Single Responsibility Principle (SRP):** Six dedicated services manage independent domains. Route handlers inject only the service they require.
2. **Zero Database Leaks:** Higher layers (handlers/services) never import `server/internal/platform/db/generated` directly. The `AuthRepository` interface abstracts all database access.
3. **Transport Agnostic:** Service logic does not depend on HTTP frameworks, headers, or cookies. It accepts domain inputs and returns domain outputs.
4. **Defense in Depth:** Timing-attack mitigations, pessimistic row-level locking on OTPs, and refresh token replay theft detection are enforced within the service layer.

---

## 2. Directory Structure

```text
server/internal/auth/
├── config/
│   └── config.go              # AuthConfig structs & default production config
├── autherr/
│   └── errors.go              # Domain sentinel errors (package autherr)
├── httperr/
│   └── classify.go            # Sentinel-error -> HTTP mapping (Classify, Fail, InvalidBody)
├── model/
│   └── model.go               # Shared structs (TokenPair, UserProfile, ProfileInput, InvitationDetails, Mailer)
├── handler/
│   ├── auth_handler.go        # HTTP handlers (Echo), envelope-migrated
│   ├── auth_handler_test.go   # Handler unit tests
│   ├── request.go             # Request DTOs
│   └── response.go            # Success-envelope builders
├── middleware/
│   ├── middleware.go          # RequireAuth, LoadUser, RequireRole, GetUserID
│   └── middleware_test.go     # Middleware unit tests
├── repository/
│   ├── repository.go          # AuthRepository interface & SQLC implementation
│   ├── mapper.go              # db -> domain mapping (MapToUserProfile, TextToPtr)
│   ├── repository_test.go     # Repository unit tests (mocked DB querier)
│   └── withtx_test.go         # Transaction-boundary tests
├── token/
│   ├── jwt.go                 # HS256 JWT generation and claim verification
│   └── tokens.go              # CSPRNG tokens, SHA-256 hasher, 6-digit OTP generator
├── hashing/
│   ├── password.go            # Bcrypt hashing & constant-time dummy comparisons
│   └── password_test.go       # Hashing unit tests
├── validate/
│   ├── validate.go            # Input checks (email, password, username, profile name, avatar URL)
│   └── validate_test.go       # Validation unit tests
└── service/
    ├── oauth/
    │   ├── google.go          # Google ID Token validator
    │   └── provider.go        # Provider interface & Identity model
    ├── issue_token.go             # issueTokenPair (JWT + persisted refresh token)
    ├── invitation_service.go      # InvitationService (Invite, GetInvitation, RevokeInvitation)
    ├── invitation_service_test.go # Invitation unit tests
    ├── mock_repository_test.go    # In-memory mock AuthRepository, Mailer, and OAuth provider
    ├── password_service.go        # PasswordService (ForgotPassword, ResetPassword, ChangePassword)
    ├── password_service_test.go   # Password unit tests
    ├── profile_service.go         # ProfileService (GetProfile, UpdateProfile, DeleteProfile, CheckUsernameAvailable)
    ├── profile_service_test.go    # Profile unit tests
    ├── services.go                # Aggregated Services container & NewServices constructor
    ├── session_service.go         # SessionService (Login, Refresh, Logout, OAuthLogin)
    ├── session_service_test.go    # Session unit tests (timing attack dummy compare, replay detection)
    ├── signup_service.go          # SignupService (Signup, VerifyEmail, ResendOTP, AcceptInvite)
    ├── signup_service_test.go     # Signup unit tests
    ├── user_admin_service.go      # UserAdminService (SuspendUser, UnsuspendUser, DeactivateUser, DeleteUser)
    └── user_admin_service_test.go # Admin unit tests
```

Platform-wide packages live under `server/internal/platform/` (`config`, `db/generated`, `mailer`, `httpserver`, `apiresponse`).

---

## 3. Account Lifecycle & State Machine

```
                       ┌──────────────────────┐
                       │  Invite Issued       │
                       └──────────┬───────────┘
                                  │ AcceptInvite()
                                  ▼
┌──────────────────┐  Signup()  ┌──────────────────────┐
│ RegistrationOpen ├───────────►│ Pending Verification │
└──────────────────┘            └──────────┬───────────┘
                                           │ VerifyEmail()
                                           ▼
                                ┌──────────────────────┐
                                │       ACTIVE         │◄────────┐
                                └─────┬──────┬─────────┘         │
                                      │      │                   │
                  SuspendUser()       │      │ DeactivateUser()  │ UnsuspendUser()
                  (Tokens Revoked)    │      │ (Tokens Revoked)  │
                                      ▼      ▼                   │
                          ┌─────────────┐  ┌───────────────┐     │
                          │  SUSPENDED  │  │  DEACTIVATED  │     │
                          └──────┬──────┘  └───────┬───────┘     │
                                 │                 │             │
                                 └─────────────────┴─────────────┘
                                           DeleteUser()
                                                │
                                                ▼
                                         ┌─────────────┐
                                         │   DELETED   │
                                         └─────────────┘
```

- **Unverified (`email_verified_at IS NULL`):** Created when `VerificationConfig.Required = true`. Cannot log in with email/password until verified via `VerifyEmail`.
- **Active (`email_verified_at IS NOT NULL`, `is_suspended = false`, `is_deactivated = false`):** Fully functional account.
- **Suspended (`is_suspended = true`):** Blocked from logging in or refreshing tokens (`ErrAccountSuspended`). All active sessions are immediately revoked upon suspension. Can be restored via `UnsuspendUser`.
- **Deactivated (`is_deactivated = true`):** Blocked from logging in or refreshing tokens (`ErrAccountDeactivated`). All sessions are revoked.
- **Deleted:** Hard deletion of refresh tokens, profile, and user records.

---

## 4. Security Architecture

### 4.1. Bcrypt Password Hashing & Side-Channel Mitigation
- Passwords are hashed using bcrypt with a configurable work factor (default: 12).
- **Anti-Enumeration on Login (`AUTH-17`):** When a user enters an unregistered email, the system executes a dummy bcrypt comparison (`DummyPasswordCompare`) against a precomputed static hash. This eliminates response-time discrepancy between existing and non-existing accounts, thwarting timing attacks.
- **Anti-Enumeration on Password Reset:** `ForgotPassword` returns `nil` silently if the email does not exist, belongs to an OAuth-only user, or is suspended.

### 4.2. Pessimistic Row Locking for OTP Verification
To eliminate race conditions (such as multiple simultaneous requests brute-forcing the same OTP before attempts increment), `GetValidOTPForUpdate` locks the active row using PostgreSQL `SELECT ... FOR UPDATE`.

### 4.3. Refresh Token Rotation & Theft Detection (`AUTH-14` / `AUTH-15`)
- When `SessionConfig.RotateRefreshToken = true`, presenting a refresh token invalidates it and issues a brand-new token pair.
- **Theft Detection:** If a revoked refresh token is presented again (indicating a potential token replay or stolen session), the service invokes `RevokeAllUserRefreshTokens(ctx, tokenRecord.UserID)`. All active sessions for that user across all devices are terminated immediately.

### 4.4. Immediate Session Termination on Sensitive Operations
All active refresh tokens for a user are revoked automatically when:
1. The user changes their password (`ChangePassword`).
2. The user resets their password (`ResetPassword`).
3. An administrator suspends the account (`SuspendUser`).
4. An administrator deactivates the account (`DeactivateUser`).
5. A refresh token replay is detected (`Refresh`).

---

## 5. Configuration Reference (`AuthConfig`)

`AuthConfig` allows any product or agency project to tailor auth behavior without modifying domain code:

```go
type AuthConfig struct {
    Registration RegistrationConfig
    Verification VerificationConfig
    Session      SessionConfig
    Password     PasswordConfig
    Profile      ProfileConfig
    OAuth        OAuthConfig
}
```

| Section | Setting | Default | Description |
|---|---|---|---|
| **Registration** | `Mode` | `RegistrationModeOpen` | `RegistrationModeOpen` allows public signup; `RegistrationModeInviteOnly` requires an invitation token. |
| | `EnablePasswordRegistration` | `true` | Allows email and password registration. |
| | `EnableOAuthRegistration` | `true` | Allows account creation via OAuth providers. |
| **Verification** | `Required` | `true` | When true, new signups require OTP verification before first login. |
| | `OTPTTL` | `15 * time.Minute` | Lifetime of 6-digit verification and password-reset OTPs. |
| | `MaxOTPAttempts` | `5` | Maximum failed verification attempts before the OTP is invalidated. |
| | `ResendCooldown` | `60 * time.Second` | Minimum delay before a new OTP can be requested for the same purpose. |
| **Session** | `AccessTokenTTL` | `15 * time.Minute` | HS256 JWT access token expiration duration. |
| | `RefreshTokenTTL` | `30 * 24 * time.Hour` | Cryptographic refresh token expiration duration (30 days). |
| | `RotateRefreshToken` | `true` | Rotates refresh token on every refresh call; detects token replay. |
| **Password** | `EnablePasswordLogin` | `true` | Enables authentication via email/password. |
| | `MinLength` | `8` | Minimum character length for passwords (max is 72 due to bcrypt). |
| | `BcryptCost` | `12` | Bcrypt hashing work factor. |
| **Profile** | `EnableName` | `true` | Stores `first_name` and `last_name`. |
| | `EnableUsername` | `true` | Enforces unique `username` across profiles. |
| | `EnableDisplayName` | `true` | Allows user-customized `display_name`. |
| **OAuth** | `Enabled` | `true` | Enables third-party identity providers. |
| | `Providers` | `map[string]OAuthProvider` | Registered providers (`google`, etc.). |

---

## 6. Service API Reference

### 6.1. `SignupService` (`signup_service.go`)
- `Signup(ctx, email, password, profileInput) (*TokenPair, error)`
  Creates a new account. If `Verification.Required` is true, returns `(nil, nil)` and sends an OTP.
- `VerifyEmail(ctx, email, code) (*TokenPair, error)`
  Verifies the signup OTP code with row locking, marks email verified, and returns the initial session token pair.
- `ResendOTP(ctx, email) error`
  Generates and dispatches a fresh signup verification OTP subject to `ResendCooldown`.
- `AcceptInvite(ctx, rawToken, password, profileInput) (*TokenPair, error)`
  Validates invite token, creates user & profile with auto-verified email, marks invite accepted, and logs user in.

### 6.2. `SessionService` (`session_service.go`)
- `Login(ctx, email, password) (*TokenPair, error)`
  Authenticates email and password. Protected by `DummyPasswordCompare` on missing users.
- `Refresh(ctx, rawRefreshToken) (*TokenPair, error)`
  Validates refresh token, executes rotation, and detects token theft/replay.
- `Logout(ctx, rawRefreshToken) error`
  Revokes the specified refresh token session.
- `OAuthLogin(ctx, providerName, idToken) (*TokenPair, error)`
  Validates provider token (e.g. Google sub/email/name), connects or provisions user account, marks email verified, and issues session tokens.

### 6.3. `PasswordService` (`password_service.go`)
- `ForgotPassword(ctx, email) error`
  Initiates a reset flow. Silent on missing/OAuth accounts to prevent user enumeration.
- `ResetPassword(ctx, email, code, newPassword) error`
  Verifies reset OTP with row locking, updates password, auto-verifies email, and revokes all active sessions.
- `ChangePassword(ctx, userID, oldPassword, newPassword) error`
  Authenticated flow. Validates old password with constant-time check, updates password, and revokes all sessions.

### 6.4. `InvitationService` (`invitation_service.go`)
- `Invite(ctx, invitedBy, email) error`
  Creates a single-use 7-day invitation token and emails the invitee.
- `GetInvitation(ctx, rawToken) (*InvitationDetails, error)`
  Returns invitation metadata for UI display without redeeming the token.
- `RevokeInvitation(ctx, invitationID) error`
  Deletes an outstanding invitation.

### 6.5. `ProfileService` (`profile_service.go`)
- `GetProfile(ctx, userID) (*UserProfile, error)`
  Retrieves user and profile information.
- `UpdateProfile(ctx, userID, profileInput) (*UserProfile, error)`
  Updates profile fields and enforces username uniqueness.
- `CheckUsernameAvailable(ctx, username) (bool, error)`
  Returns true if the username is free to register.
- `DeleteProfile(ctx, userID) error`
  Deletes the profile record while retaining user credentials.

### 6.6. `UserAdminService` (`user_admin_service.go`)
- `SuspendUser(ctx, userID) error`
  Marks user suspended and immediately revokes all active sessions.
- `UnsuspendUser(ctx, userID) error`
  Restores user status to active.
- `DeactivateUser(ctx, userID) error`
  Marks user deactivated and revokes all active sessions.
- `DeleteUser(ctx, userID) error`
  Cascades deletion of refresh tokens, profile, and user account.

### 6.7. Aggregate Container (`services.go`)
- `NewServices(repo, config, jwtIssuer, mailer) *Services`
  Constructs all 6 services with shared dependencies for server startup in `main.go`.
