# Olona — HTTP Response Contract

> **Status: agreed, not yet implemented.**
> Every handler in `server/` currently returns an older, flat shape — `{"error": "..."}`,
> unwrapped payloads, `200` with a prose `message`. This document is the **target**.
> Nothing here describes live behaviour until the handlers are migrated.
>
> Auth is deliberately staying on the flat shape for now so it isn't disturbed mid-flight.
> Migrate when the product work begins.

---

## 1. Why an envelope

Three problems the flat shape can't solve:

1. **Prose is not machine-readable.** Today `handleError` emits `err.Error()`. A client that needs to distinguish "invite expired" from "invite already used" has to string-match English.
2. **Nothing separates payload from metadata.** Pagination, or an error, has nowhere to live except inside the resource.
3. **Success and failure look the same.** Both are a bare JSON object; the client can only tell them apart by checking the HTTP status.

One envelope, applied everywhere, fixes all three.

---

## 2. The three shapes

### 2.1. Success — single resource

```json
{
  "data": {
    "id": "usr_123",
    "email": "a@b.com"
  }
}
```

`HTTP 200`

### 2.2. Success — collection

```json
{
  "data": [
    { "id": "usr_123" },
    { "id": "usr_456" }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 137
  }
}
```

`HTTP 200`

`meta` appears **only** on collections. Omit it otherwise.

### 2.3. Error

```json
{
  "error": {
    "code": "validation_failed",
    "message": "Check the highlighted fields.",
    "details": [
      { "field": "username", "code": "taken" },
      { "field": "password", "code": "too_short" }
    ]
  }
}
```

`HTTP 400`

`details` is **optional**. Omit it when nothing is field-specific.

### 2.4. Error — auth failures carry no `details`, ever

```json
{
  "error": {
    "code": "invalid_credentials",
    "message": "Email or password is incorrect."
  }
}
```

`HTTP 401`

> A `details` array saying `{"field": "email", "code": "not_found"}` would tell an
> attacker which emails are registered — precisely what this message exists to hide.
> **Never attach `details` to an auth failure.**

---

## 3. Rules

| Rule | |
| --- | --- |
| Exactly one of `data` / `error` | Never both. Never neither. |
| On error, `data` is **absent** | Not `null`. |
| On success, `error` is **absent** | |
| `meta` only on collections | Omit otherwise. |
| `details` only on validation errors | Never on auth failures. |
| `code` is a closed set | §5. The frontend branches on this. |
| `message` is for humans | Toasts and display. **Never branch on it.** |
| Keys are `snake_case` | `per_page`, not `perPage`. |

### The one rule that matters

> **Exactly one of `data` / `error` is present. Never both. Never neither.**

That's what makes the envelope work. If it isn't enforced, someone eventually ships a
response containing both, and every client's `if` chain has to handle it forever.

---

## 4. Emptiness and null

"There's nothing here" has four distinct meanings. Each gets a different answer.

### 4.1. Empty collection → `[]`

```json
{ "data": [], "meta": { "page": 1, "per_page": 20, "total": 0 } }
```

- Always an **empty array**, never `null`, never `""`.
- `""` is a string — it is not a collection and will break a typed client.
- This is a **success**, not a 404. The query ran; there were no rows.
- `meta` is still present, so the client can tell "empty page" from "no pagination".

### 4.2. Single resource not found → `404`, `data` absent

```json
{ "error": { "code": "not_found", "message": "That item doesn't exist." } }
```

Not `{"data": null}`. Absent.

### 4.3. Void operation → `204`, no body

`DELETE /users/me`, `POST /auth/logout`, and anything else with nothing to return
respond `204 No Content` with **no body at all** — not `{"data": null}`.

### 4.4. A null *field* inside `data` → keep it

```json
{
  "data": {
    "id": "usr_123",
    "avatar_url": null
  }
}
```

Nulls **inside** `data` are meaningful and must survive. "This user has no avatar" is
information. This rule applies only to `data` itself, never to its fields.

### Summary

| Situation | Response |
| --- | --- |
| Query returned no rows | `200` · `"data": []` |
| Single resource missing | `404` · `error`, no `data` |
| Nothing to return (delete, logout) | `204` · no body |
| Field is genuinely absent | `200` · `"field": null` inside `data` |
| Never | `"data": null`, `"data": ""` |

---

## 5. Error codes

A closed set. Derived from the sentinel errors currently mapped in
`auth_handler.go` → `handleError`. **This table is the contract** — adding an error
means adding a row here in the same commit.

### `400` — request is malformed

| `code` | Meaning |
| --- | --- |
| `validation_failed` | One or more fields failed validation; see `details`. |
| `invalid_body` | Body missing, malformed, or unparseable. |
| `password_too_short` | |
| `password_too_long` | |
| `password_same` | New password matches the current one. |
| `invalid_email` | |
| `invalid_username` | |
| `invalid_profile_name` | |
| `invalid_avatar_url` | |
| `missing_parameter` | A required query parameter was absent. |

### `401` — not authenticated, or deliberately ambiguous

| `code` | Meaning |
| --- | --- |
| `invalid_credentials` | Wrong email or password. **Deliberately ambiguous.** |
| `invalid_token` | Access token missing, malformed, or expired. |
| `refresh_token_invalid` | |
| `refresh_token_reused` | Replay detected; the whole family is revoked. |
| `email_not_verified` | |
| `otp_incorrect` | |
| `otp_expired` | |
| `otp_not_found` | |
| `invitation_not_found` | |
| `invitation_expired` | |
| `invitation_already_accepted` | |
| `oauth_account` | Account exists via a provider; password login is not available. |
| `oauth_provider_not_supported` | |
| `oauth_token_invalid` | |
| `password_login_disabled` | |
| `registration_disabled` | Public registration is closed. |

> The three `invitation_*` codes are why this envelope exists. All three are `401`
> today with nothing but English to tell them apart. The frontend routes each to a
> different screen.

### `403` — authenticated, but not allowed

| `code` | Meaning |
| --- | --- |
| `account_suspended` | |
| `account_deactivated` | |

### `404`

| `code` | Meaning |
| --- | --- |
| `not_found` | Generic. Never reveal *which* resource — see `AUTH-04`. |

### `409` — conflicts with existing state

| `code` | Meaning |
| --- | --- |
| `email_already_registered` | |
| `username_taken` | |

### `429`

| `code` | Meaning |
| --- | --- |
| `otp_max_attempts` | |
| `otp_cooldown` | Resend cooldown still active. |

### `500`

| `code` | Meaning |
| --- | --- |
| `internal_error` | Always this. Never leak the underlying error. |

### Field-level codes (`details[].code`)

Smaller, reusable set: `required`, `taken`, `too_short`, `too_long`, `invalid`, `mismatch`.

```json
{ "field": "username", "code": "taken" }
```

---

## 6. Exemptions

**`GET /health` is exempt.** It returns the bare `{"status": "ok"}` and must keep doing so.

Health checks are consumed by infrastructure (the Space's own probe, load balancers),
not by the app. Wrapping it in an envelope adds a failure mode to the thing whose only
job is to report whether the process is alive.

---

## 7. Migration notes

When this is implemented, it is a **breaking change** for every client.

**Backend surface:**

| | |
| --- | --- |
| `c.JSON` call sites | 39 — 1 health (exempt), 3 middleware, 35 handlers |
| Swaggo annotation lines | 59 across 14 endpoints |
| Sentinel errors needing a code | 29 |
| Generated docs | `docs/docs.go`, `swagger.json`, `swagger.yaml` |

**Four things that aren't mechanical:**

1. **Swaggo cannot express generics.** A `Response[T]` type will not generate a usable
   schema. Concrete named wrapper types are required per payload.
2. **`omitempty` does not work on structs.** `data` must be a pointer (or `any`) to be
   genuinely absent rather than `null`.
3. **29 errors need 29 codes** — a mapping table that must be kept in sync with §5.
4. **`details[]` needs information that doesn't exist yet.** `ErrPasswordTooShort` is a
   single sentinel with no field attached. Emitting `{"field": "password", ...}` requires
   either a `ValidationError` type carrying field and code, or a sentinel→field lookup.
   This is the largest decision in the migration and it reaches into the service layer.

**Behaviour changes, not just shape:**

- Several endpoints currently return `200` with a prose `message`
  (`"profile deleted successfully"`, `"password changed successfully"`) — these become
  `204` with no body.

**Sequencing:** do it once, at the start of product work, before client code depends on
the shape. Migrating after that means every call site on both sides changes twice.
