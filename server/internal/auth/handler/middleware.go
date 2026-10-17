package handler

import (
	"net/http"
	"strings"

	"server/internal/auth/model"
	"server/internal/auth/repository"
	"server/internal/auth/utils"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// RequireAuth is a middleware that enforces JWT authentication for protected routes.
// It extracts the token from the Authorization header, validates it using the JWTIssuer,
// and injects the user_id into the Echo context.
func RequireAuth(issuer *utils.JWTIssuer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing authorization header"})
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid authorization header format"})
			}

			tokenString := parts[1]
			claims, err := issuer.Validate(tokenString)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid or expired token"})
			}

			// Store user ID in context for downstream handlers
			c.Set("user_id", claims.UserID)
			return next(c)
		}
	}
}

// LoadUser fetches the account behind a validated token and puts it in
// context for later middleware. Must run after RequireAuth, which is what
// sets "user_id".
func LoadUser(repo repository.AuthRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get the id RequireAuth stored
			id := GetUserID(c)

			// Fetch the row
			user, err := repo.GetUserByID(c.Request().Context(), id)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "user does not exist"})
			}

			// If suspended or deactivated
			if user.IsSuspended {
				return c.JSON(http.StatusForbidden, ErrorResponse{Error: "account suspended"})
			}

			if user.IsDeactivated {
				return c.JSON(http.StatusForbidden, ErrorResponse{Error: "account deactivated"})
			}

			// Store the account so RequireRole can read it
			c.Set("account", model.Actor{
				UserID:        id,
				Role:          model.Role(user.Role),
				IsSuspended:   user.IsSuspended,
				IsDeactivated: user.IsDeactivated,
			})
			return next(c)
		}
	}

}

// RequireRole refuses the request unless the account's role is one of allowed.
// Must run after LoadUser.
func RequireRole(allowed ...model.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Read the account from context
			account, ok := c.Get("account").(model.Actor)

			// Is their role in `allowed`?
			if !ok {
				return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "not authenticated"})
			}

			for _, role := range allowed {
				if account.Role == role {
					return next(c)
				}
			}

			return c.JSON(http.StatusForbidden, ErrorResponse{Error: "not allowed"})
		}
	}
}

// GetUserID extracts the user ID from the Echo context.
// It panics if the ID is missing, which is safe because this should only be called
// in handlers protected by RequireAuth.
func GetUserID(c echo.Context) uuid.UUID {
	id, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		panic("user_id missing from context; did you forget the RequireAuth middleware?")
	}
	return id
}
