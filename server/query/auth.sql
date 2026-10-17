-- ===================================
-- USER
-- ===================================

-- name: CreateUser :one
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $1
WHERE id = $2;

-- name: MarkEmailVerified :exec
UPDATE users
SET email_verified_at = now()
WHERE id = $1;

-- name: SuspendUser :exec
UPDATE users
SET is_suspended = true
WHERE id = $1;

-- name: UnsuspendUser :exec
UPDATE users
SET is_suspended = false
WHERE id = $1;

-- name: DeactivateUser :exec
UPDATE users
SET is_deactivated = true
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- ===================================
-- PROFILE
-- ===================================

-- name: CreateProfile :one
INSERT INTO profiles (
  user_id,
  first_name,
  last_name,
  username,
  display_name,
  profile_url,
  avatar_url
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetProfileByUserID :one
SELECT * FROM profiles
WHERE user_id = $1;

-- name: UpdateProfile :one
UPDATE profiles
SET 
  first_name = COALESCE(sqlc.narg('first_name'), first_name),
  last_name = COALESCE(sqlc.narg('last_name'), last_name),
  username = COALESCE(sqlc.narg('username'), username),
  display_name = COALESCE(sqlc.narg('display_name'), display_name),
  profile_url = COALESCE(sqlc.narg('profile_url'), profile_url),
  avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url)
WHERE user_id = sqlc.arg('user_id')
RETURNING *;

-- name: CheckUsernameExists :one
SELECT EXISTS(
  SELECT 1 FROM profiles WHERE username = $1
);

-- name: DeleteProfile :exec
DELETE FROM profiles
WHERE user_id = $1;

-- ===================================
-- OAUTH
-- ===================================

-- name: GetOAuthConnection :one
SELECT * FROM oauth_connections
WHERE provider = $1 AND provider_user_id = $2;

-- name: GetUserByOAuthProvider :one
SELECT u.* FROM users u
JOIN oauth_connections o ON u.id = o.user_id
WHERE o.provider = $1 AND o.provider_user_id = $2;

-- name: CreateOAuthConnection :one
INSERT INTO oauth_connections (user_id, provider, provider_user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteOAuthConnection :exec
DELETE FROM oauth_connections
WHERE user_id = $1 AND provider = $2;

-- name: GetOAuthProvidersForUser :many
SELECT provider FROM oauth_connections
WHERE user_id = $1;

-- ===================================
-- REFRESH TOKENS
-- ===================================

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens
WHERE id = $1;

-- ===================================
-- OTP
-- ===================================

-- name: CreateOTP :one
INSERT INTO otp_codes (user_id, code_hash, purpose, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetValidOTP :one
SELECT * FROM otp_codes
WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
ORDER BY created_at DESC
LIMIT 1;

-- name: GetValidOTPForUpdate :one
SELECT * FROM otp_codes
WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
ORDER BY created_at DESC
LIMIT 1
FOR UPDATE;

-- name: IncrementOTPAttempts :one
UPDATE otp_codes
SET attempts = attempts + 1
WHERE id = $1
RETURNING attempts;

-- name: MarkOTPUsed :exec
UPDATE otp_codes
SET used_at = now()
WHERE id = $1;

-- name: DeleteOTP :exec
DELETE FROM otp_codes
WHERE id = $1;

-- ===================================
-- INVITATIONS
-- ===================================

-- name: CreateInvitation :one
INSERT INTO invitations (email, token_hash, invited_by, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetInvitationByID :one
SELECT * FROM invitations
WHERE id = $1;

-- name: GetInvitationByToken :one
SELECT * FROM invitations
WHERE token_hash = $1 AND accepted_at IS NULL AND expires_at > now();

-- name: MarkInvitationAccepted :exec
UPDATE invitations
SET accepted_at = now()
WHERE id = $1;

-- name: DeleteInvitation :exec
DELETE FROM invitations
WHERE id = $1;
