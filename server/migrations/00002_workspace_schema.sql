-- +goose Up
-- +goose StatementBegin
-- 1. Custom ENUM types for state machines and tiers
CREATE TYPE workspace_status AS ENUM (
    'active',
    'suspended',
    'archive_requested',
    'archived',
    'delete_requested',
    'pending_deletion',
    'deleted'
);
CREATE TYPE workspace_tier AS ENUM (
    'owner',
    'admin',
    'member',
    'viewer'
);
CREATE TYPE workspace_member_status AS ENUM ('active', 'removed');
CREATE TYPE workspace_request_type AS ENUM (
    'archive',
    'delete',
    'transfer_ownership',
    'move'
);
CREATE TYPE workspace_request_status AS ENUM (
    'pending',
    'approved',
    'rejected',
    'cancelled',
    'expired'
);
-- 2. Workspaces table
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id UUID REFERENCES workspaces(id) ON DELETE RESTRICT,
    root_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE RESTRICT,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status workspace_status NOT NULL DEFAULT 'active',
    single_principal BOOLEAN NOT NULL DEFAULT false,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at TIMESTAMPTZ,
    suspended_at TIMESTAMPTZ,
    suspended_by UUID REFERENCES users(id),
    delete_after TIMESTAMPTZ
);
-- Self-referential root_id deferral or self-link is needed on root insert.
-- We allow parent_id NULL for root. root_id can point to self.
-- We index parent_id and root_id for fast tree lookup.
CREATE INDEX idx_workspaces_parent_id ON workspaces(parent_id);
CREATE INDEX idx_workspaces_root_id ON workspaces(root_id);
CREATE INDEX idx_workspaces_status ON workspaces(status);
-- Sibling name uniqueness: name must be unique among children of the same parent
CREATE UNIQUE INDEX idx_workspaces_sibling_name ON workspaces(
    COALESCE(
        parent_id,
        '00000000-0000-0000-0000-000000000000'::uuid
    ),
    name
)
WHERE status != 'deleted';
-- Trigger for workspaces.updated_at
CREATE TRIGGER workspaces_updated_at BEFORE
UPDATE ON workspaces FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- 3. Workspace Closure table (for fast ancestry & oversight resolution)
CREATE TABLE workspace_closure (
    ancestor_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    descendant_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    depth INT NOT NULL,
    PRIMARY KEY (ancestor_id, descendant_id)
);
CREATE INDEX idx_workspace_closure_descendant ON workspace_closure(descendant_id);
-- 4. Workspace Members table
CREATE TABLE workspace_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tier workspace_tier NOT NULL,
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    status workspace_member_status NOT NULL DEFAULT 'active',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    invited_by UUID REFERENCES users(id)
);
-- Constraint: Unique active membership per (workspace_id, user_id)
CREATE UNIQUE INDEX idx_workspace_members_active_user ON workspace_members(workspace_id, user_id)
WHERE status = 'active';
-- Constraint: Exactly one active owner per workspace
CREATE UNIQUE INDEX idx_workspace_members_single_owner ON workspace_members(workspace_id)
WHERE tier = 'owner'
    AND status = 'active';
CREATE INDEX idx_workspace_members_user ON workspace_members(user_id);
-- 5. Workspace Invitations table
CREATE TABLE workspace_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    tier workspace_tier NOT NULL,
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    invited_by UUID NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_invitation_no_owner CHECK (tier != 'owner')
);
CREATE INDEX idx_workspace_invitations_workspace ON workspace_invitations(workspace_id);
CREATE INDEX idx_workspace_invitations_token_hash ON workspace_invitations(token_hash);
CREATE INDEX idx_workspace_invitations_email ON workspace_invitations(email);
-- 6. Workspace Requests (Two-key control approval queue)
CREATE TABLE workspace_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    type workspace_request_type NOT NULL,
    status workspace_request_status NOT NULL DEFAULT 'pending',
    requested_by UUID NOT NULL REFERENCES users(id),
    reason TEXT,
    target_user_id UUID REFERENCES users(id),
    new_parent_id UUID REFERENCES workspaces(id),
    decided_by UUID REFERENCES users(id),
    decided_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    sole_approver BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_decided_by_different CHECK (
        decided_by IS NULL
        OR decided_by != requested_by
        OR sole_approver = true
    )
);
CREATE INDEX idx_workspace_requests_workspace ON workspace_requests(workspace_id);
CREATE INDEX idx_workspace_requests_status ON workspace_requests(status);
CREATE INDEX idx_workspace_requests_requested_by ON workspace_requests(requested_by);
-- 7. Workspace Audit Log (Append-only immutable audit trail)
CREATE TABLE workspace_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(100) NOT NULL,
    target_id UUID,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_workspace_audit_workspace ON workspace_audit(workspace_id, created_at DESC);
CREATE INDEX idx_workspace_audit_actor ON workspace_audit(actor_id);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS workspace_audit CASCADE;
DROP TABLE IF EXISTS workspace_requests CASCADE;
DROP TABLE IF EXISTS workspace_invitations CASCADE;
DROP TABLE IF EXISTS workspace_members CASCADE;
DROP TABLE IF EXISTS workspace_closure CASCADE;
DROP TABLE IF EXISTS workspaces CASCADE;
DROP TYPE IF EXISTS workspace_request_status;
DROP TYPE IF EXISTS workspace_request_type;
DROP TYPE IF EXISTS workspace_member_status;
DROP TYPE IF EXISTS workspace_tier;
DROP TYPE IF EXISTS workspace_status;
-- +goose StatementEnd