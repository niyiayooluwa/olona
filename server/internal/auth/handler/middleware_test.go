package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"server/internal/auth/model"
	"server/internal/auth/repository"
	db "server/internal/db/generated"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// stubRepository stands in for repository.AuthRepository.
//
// It embeds the interface rather than reimplementing it, so every method
// except GetUserByID dispatches to a nil interface and panics. That is the
// behaviour this file wants: LoadUser is entitled to read an account row and
// nothing else, and a panic names whatever else it started calling.
type stubRepository struct {
	repository.AuthRepository
	user db.User
	err  error
}

func (s *stubRepository) GetUserByID(context.Context, uuid.UUID) (db.User, error) {
	return s.user, s.err
}

// chain composes middleware the way Echo does: the first argument runs first.
// Tests that care about ordering use this instead of registering a real route.
func chain(mws ...echo.MiddlewareFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		h := next
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}
		return h
	}
}

// runMiddleware drives one middleware (or a chain) the way Echo would: it
// builds a context around a recorder, installs a terminal handler, and reports
// whether that handler was reached.
//
// Returning the context matters — LoadUser's whole job is what it leaves *in*
// the context, so a test that could not read it back would only be checking
// half of the contract.
func runMiddleware(t *testing.T, mw echo.MiddlewareFunc, setup func(c echo.Context)) (echo.Context, *httptest.ResponseRecorder, bool) {
	t.Helper()

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/v1/admin/ping", nil), rec)
	if setup != nil {
		setup(c)
	}

	reached := false
	next := func(c echo.Context) error {
		reached = true
		return c.NoContent(http.StatusOK)
	}

	if err := mw(next)(c); err != nil {
		t.Fatalf("middleware returned %v; it must answer with c.JSON, not a bare error", err)
	}
	return c, rec, reached
}

// setUserID performs the single context write RequireAuth is responsible for.
// Tests that start downstream of it use this rather than minting a real token,
// which keeps them about the middleware under test and not about JWT handling.
func setUserID(id uuid.UUID) func(echo.Context) {
	return func(c echo.Context) { c.Set("user_id", id) }
}

// TestLoadUserPutsActorInContext is the happy path, and it also pins the
// boundary between the database row and the HTTP layer: the context carries a
// model.Actor, never a db.User, so handlers never have to reach for the
// generated package.
func TestLoadUserPutsActorInContext(t *testing.T) {
	id := uuid.New()
	repo := &stubRepository{user: db.User{ID: id, Role: "staff"}}

	c, rec, reached := runMiddleware(t, LoadUser(repo), setUserID(id))

	if !reached {
		t.Fatalf("terminal handler not reached; status %d body %s", rec.Code, rec.Body.String())
	}

	raw := c.Get("account")
	actor, ok := raw.(model.Actor)
	if !ok {
		t.Fatalf("account in context is %T, want model.Actor", raw)
	}
	if actor.UserID != id {
		t.Errorf("actor.UserID = %s, want %s", actor.UserID, id)
	}
	if actor.Role != model.RoleStaff {
		t.Errorf("actor.Role = %q, want %q", actor.Role, model.RoleStaff)
	}
}

// TestLoadUserRefusesBlockedOrMissingAccounts covers the three ways LoadUser
// can stop a request before any handler runs.
//
// The status codes are the point, and they are two different claims. A
// suspended or deactivated account holds a valid token, so it is 403 — the
// credential is good, the account may not act. A user id with no row behind it
// is 401, because there is nothing to authorise against.
//
// None of these may be a 500. A bare error returned from middleware reaches
// Echo's default handler, which knows nothing about these cases: it would
// relabel a deliberate refusal as a server fault and leak the message while
// doing it. That regression is silent, so it is locked down here.
func TestLoadUserRefusesBlockedOrMissingAccounts(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name string
		repo *stubRepository
		want int
	}{
		{
			name: "suspended",
			repo: &stubRepository{user: db.User{ID: id, Role: "client", IsSuspended: true}},
			want: http.StatusForbidden,
		},
		{
			name: "deactivated",
			repo: &stubRepository{user: db.User{ID: id, Role: "client", IsDeactivated: true}},
			want: http.StatusForbidden,
		},
		{
			name: "no such user",
			repo: &stubRepository{err: repository.ErrNotFound},
			want: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, rec, reached := runMiddleware(t, LoadUser(tt.repo), setUserID(id))

			if reached {
				t.Fatal("terminal handler ran; the request should have been stopped")
			}
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), `"error"`) {
				t.Errorf("body %s is not the error envelope", rec.Body.String())
			}
		})
	}
}

// TestRequireRoleAllowsListedRole covers the allow path, including the varargs
// form. A route opened to admin and staff has to admit staff, not just the
// first role listed — an accidental "only check allowed[0]" would pass a
// single-role test and quietly lock staff out of everything.
func TestRequireRoleAllowsListedRole(t *testing.T) {
	for _, role := range []model.Role{model.RoleAdmin, model.RoleStaff} {
		t.Run(string(role), func(t *testing.T) {
			_, rec, reached := runMiddleware(t,
				RequireRole(model.RoleAdmin, model.RoleStaff),
				func(c echo.Context) {
					c.Set("account", model.Actor{UserID: uuid.New(), Role: role})
				})

			if !reached {
				t.Fatalf("role %q was refused; status %d body %s", role, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestRequireRoleRefusesUnlistedRole is the denial the requirements depend on:
// a client reaching an admin route gets 403, not 404. The caller is
// authenticated and the route exists — they are simply not allowed. 404 is
// reserved for the other case, where the caller has the right role but the
// resource belongs to another tenant.
func TestRequireRoleRefusesUnlistedRole(t *testing.T) {
	_, rec, reached := runMiddleware(t, RequireRole(model.RoleAdmin), func(c echo.Context) {
		c.Set("account", model.Actor{UserID: uuid.New(), Role: model.RoleClient})
	})

	if reached {
		t.Fatal("client reached an admin-only handler")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestRequireRoleRefusesWithoutAccount guards the ordering contract. RequireRole
// reads what LoadUser wrote, so mounting it without LoadUser ahead of it must
// end as a 401 rather than a panic or, worse, a silent allow.
func TestRequireRoleRefusesWithoutAccount(t *testing.T) {
	_, rec, reached := runMiddleware(t, RequireRole(model.RoleAdmin), nil)

	if reached {
		t.Fatal("handler ran with no account in context")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestLoadUserRunsBeforeRequireRole pins the chain order with the case that
// makes it matter: an administrator whose access was switched off. RequireRole
// alone reads role=admin and would wave them through, so a deployment that
// mounted the two the wrong way round would hand a suspended administrator the
// keys back. LoadUser has to be the one that refuses.
func TestLoadUserRunsBeforeRequireRole(t *testing.T) {
	id := uuid.New()
	repo := &stubRepository{user: db.User{ID: id, Role: "admin", IsSuspended: true}}

	_, rec, reached := runMiddleware(t,
		chain(LoadUser(repo), RequireRole(model.RoleAdmin)),
		setUserID(id))

	if reached {
		t.Fatal("suspended admin reached the admin handler")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body: %s)", rec.Code, rec.Body.String())
	}
}
