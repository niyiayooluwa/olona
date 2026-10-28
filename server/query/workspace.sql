-- ===================================
-- WORKSPACES
-- ===================================

-- name: CreateRootWorkspace :one
INSERT INTO workspaces (id, parent_id, root_id, name, status, single_principal, created_by)
VALUES ($1, NULL, $1, $2, 'active', false, $3)
RETURNING *;

-- name: CreateChildWorkspace :one
INSERT INTO workspaces (id, parent_id, root_id, name, status, single_principal, created_by)
VALUES ($1, $2, $3, $4, 'active', $5, $6)
RETURNING *;

-- name: GetWorkspaceByID :one
SELECT * FROM workspaces
WHERE workspaces.id = $1;

-- name: UpdateWorkspaceName :one
UPDATE workspaces
SET name = $2
WHERE workspaces.id = $1
RETURNING *;

-- name: SuspendWorkspace :one
UPDATE workspaces
SET status = 'suspended',
    suspended_at = now(),
    suspended_by = $2
WHERE workspaces.id = $1
RETURNING *;

-- name: ResumeWorkspace :one
UPDATE workspaces
SET status = 'active',
    suspended_at = NULL,
    suspended_by = NULL
WHERE workspaces.id = $1
RETURNING *;

-- name: SetWorkspaceStatus :one
UPDATE workspaces
SET status = $2,
    archived_at = CASE WHEN $2 = 'archived'::workspace_status THEN now() ELSE workspaces.archived_at END,
    delete_after = CASE WHEN $2 = 'pending_deletion'::workspace_status THEN now() + interval '30 days' ELSE workspaces.delete_after END
WHERE workspaces.id = $1
RETURNING *;

-- name: CountActiveChildren :one
SELECT COUNT(*) FROM workspaces
WHERE workspaces.parent_id = $1 
  AND workspaces.status NOT IN ('archived', 'pending_deletion', 'deleted');

-- name: MoveWorkspace :one
UPDATE workspaces
SET parent_id = $2
WHERE workspaces.id = $1
RETURNING *;

-- ===================================
-- WORKSPACE CLOSURE (Cheat-Sheet)
-- ===================================

-- name: InsertRootClosure :exec
INSERT INTO workspace_closure (ancestor_id, descendant_id, depth)
VALUES ($1, $1, 0);

-- name: InsertChildClosure :exec
-- Copies all ancestors of the parent with depth+1, and adds the self row (depth 0)
INSERT INTO workspace_closure (ancestor_id, descendant_id, depth)
SELECT parent_closure.ancestor_id, $1::uuid, parent_closure.depth + 1
FROM workspace_closure AS parent_closure
WHERE parent_closure.descendant_id = $2
UNION ALL
SELECT $1::uuid, $1::uuid, 0;

-- name: IsAncestor :one
SELECT EXISTS (
    SELECT 1 FROM workspace_closure
    WHERE workspace_closure.ancestor_id = $1 
      AND workspace_closure.descendant_id = $2
);

-- name: GetAncestorAtDepth :one
SELECT workspace_closure.ancestor_id FROM workspace_closure
WHERE workspace_closure.descendant_id = $1 
  AND workspace_closure.depth = $2;

-- name: GetImmediateParentClosure :one
SELECT workspace_closure.ancestor_id FROM workspace_closure
WHERE workspace_closure.descendant_id = $1 
  AND workspace_closure.depth = 1;

-- name: DeleteSubtreeClosureForMove :exec
-- When moving subtree rooted at $1, remove paths between ancestors of $1 and descendants of $1
DELETE FROM workspace_closure AS target_closure
WHERE target_closure.descendant_id IN (
    SELECT subtree_descendant.descendant_id 
    FROM workspace_closure AS subtree_descendant 
    WHERE subtree_descendant.ancestor_id = $1
)
AND target_closure.ancestor_id IN (
    SELECT old_ancestor.ancestor_id 
    FROM workspace_closure AS old_ancestor 
    WHERE old_ancestor.descendant_id = $1 
      AND old_ancestor.ancestor_id != $1
);

-- name: RebuildClosureForMovedSubtree :exec
-- Connect new ancestors to all nodes in the moved subtree
INSERT INTO workspace_closure (ancestor_id, descendant_id, depth)
SELECT 
    new_ancestors.ancestor_id, 
    subtree_nodes.descendant_id, 
    new_ancestors.depth + subtree_nodes.depth + 1
FROM workspace_closure AS new_ancestors
CROSS JOIN workspace_closure AS subtree_nodes
WHERE new_ancestors.descendant_id = $2
  AND subtree_nodes.ancestor_id = $1;

-- ===================================
-- WORKSPACE MEMBERS & RELOCATION
-- ===================================

-- name: AddWorkspaceMember :one
INSERT INTO workspace_members (workspace_id, user_id, tier, capabilities, status, invited_by)
VALUES ($1, $2, $3, $4, 'active', $5)
RETURNING *;

-- name: GetActiveMember :one
SELECT * FROM workspace_members
WHERE workspace_members.workspace_id = $1 
  AND workspace_members.user_id = $2 
  AND workspace_members.status = 'active';

-- name: GetActiveOwner :one
SELECT * FROM workspace_members
WHERE workspace_members.workspace_id = $1 
  AND workspace_members.tier = 'owner' 
  AND workspace_members.status = 'active';

-- name: ListActiveMembers :many
SELECT 
    members.*, 
    users.email, 
    profiles.first_name, 
    profiles.last_name, 
    profiles.display_name, 
    profiles.avatar_url
FROM workspace_members AS members
JOIN users ON users.id = members.user_id
LEFT JOIN profiles ON profiles.user_id = members.user_id
WHERE members.workspace_id = $1 
  AND members.status = 'active'
ORDER BY 
    CASE members.tier 
        WHEN 'owner' THEN 1 
        WHEN 'admin' THEN 2 
        WHEN 'member' THEN 3 
        WHEN 'viewer' THEN 4 
    END, 
    members.joined_at ASC;

-- name: UpdateMemberTierAndCapabilities :one
UPDATE workspace_members
SET tier = $3,
    capabilities = $4
WHERE workspace_members.workspace_id = $1 
  AND workspace_members.user_id = $2 
  AND workspace_members.status = 'active'
RETURNING *;

-- name: RemoveMember :exec
UPDATE workspace_members
SET status = 'removed'
WHERE workspace_members.workspace_id = $1 
  AND workspace_members.user_id = $2 
  AND workspace_members.status = 'active';

-- name: TransferOwnershipInWorkspace :exec
-- Atomically demotes existing owner to admin and promotes target admin to owner
WITH demoted_owner AS (
    UPDATE workspace_members AS existing_owner
    SET tier = 'admin'
    WHERE existing_owner.workspace_id = $1 
      AND existing_owner.user_id = $2 
      AND existing_owner.tier = 'owner' 
      AND existing_owner.status = 'active'
    RETURNING existing_owner.id
)
UPDATE workspace_members AS promoted_admin
SET tier = 'owner'
WHERE promoted_admin.workspace_id = $1 
  AND promoted_admin.user_id = $3 
  AND promoted_admin.tier = 'admin' 
  AND promoted_admin.status = 'active';

-- ===================================
-- VISIBILITY & EFFECTIVE ACCESS
-- ===================================

-- name: ListVisibleWorkspacesForUser :many
-- Returns all active/accessible workspaces where user has direct membership or oversight
SELECT DISTINCT 
    workspaces.id,
    workspaces.parent_id,
    workspaces.root_id,
    workspaces.name,
    workspaces.status,
    workspaces.single_principal,
    workspaces.created_at,
    workspaces.updated_at,
    direct_member.tier AS member_tier,
    direct_member.capabilities AS member_capabilities,
    oversight_records.ancestor_id IS NOT NULL AS has_oversight
FROM workspaces
-- Direct membership match
LEFT JOIN workspace_members AS direct_member 
    ON direct_member.workspace_id = workspaces.id 
   AND direct_member.user_id = $1 
   AND direct_member.status = 'active'
-- Oversight match (user is owner/admin with oversight authority on an ancestor)
LEFT JOIN (
    SELECT DISTINCT 
        closure.descendant_id, 
        closure.ancestor_id
    FROM workspace_closure AS closure
    JOIN workspace_members AS ancestor_member 
        ON ancestor_member.workspace_id = closure.ancestor_id 
       AND ancestor_member.user_id = $1 
       AND ancestor_member.status = 'active'
       AND ancestor_member.tier IN ('owner', 'admin')
    WHERE closure.depth > 0
) AS oversight_records ON oversight_records.descendant_id = workspaces.id
WHERE (direct_member.id IS NOT NULL OR oversight_records.ancestor_id IS NOT NULL)
  AND workspaces.status != 'deleted'
ORDER BY workspaces.name ASC;

-- name: GetEffectiveAccess :one
-- Resolves whether caller has direct membership or oversight access to a workspace
SELECT 
    direct_member.tier AS direct_tier,
    direct_member.capabilities AS direct_capabilities,
    EXISTS (
        SELECT 1
        FROM workspace_closure AS closure
        JOIN workspace_members AS ancestor_member 
            ON ancestor_member.workspace_id = closure.ancestor_id 
           AND ancestor_member.user_id = $1 
           AND ancestor_member.status = 'active'
           AND ancestor_member.tier IN ('owner', 'admin')
        WHERE closure.descendant_id = $2 
          AND closure.depth > 0
    ) AS has_oversight
FROM workspaces
LEFT JOIN workspace_members AS direct_member 
    ON direct_member.workspace_id = workspaces.id 
   AND direct_member.user_id = $1 
   AND direct_member.status = 'active'
WHERE workspaces.id = $2;

-- ===================================
-- WORKSPACE INVITATIONS
-- ===================================

-- name: CreateWorkspaceInvitation :one
INSERT INTO workspace_invitations (
    workspace_id, email, tier, capabilities, token_hash, invited_by, expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetWorkspaceInvitationByToken :one
SELECT * FROM workspace_invitations
WHERE workspace_invitations.token_hash = $1 
  AND workspace_invitations.accepted_at IS NULL 
  AND workspace_invitations.revoked_at IS NULL 
  AND workspace_invitations.expires_at > now();

-- name: ListPendingInvitations :many
SELECT * FROM workspace_invitations
WHERE workspace_invitations.workspace_id = $1 
  AND workspace_invitations.accepted_at IS NULL 
  AND workspace_invitations.revoked_at IS NULL 
  AND workspace_invitations.expires_at > now()
ORDER BY workspace_invitations.created_at DESC;

-- name: MarkWorkspaceInvitationAccepted :exec
UPDATE workspace_invitations
SET accepted_at = now()
WHERE workspace_invitations.id = $1;

-- name: RevokeWorkspaceInvitation :exec
UPDATE workspace_invitations
SET revoked_at = now()
WHERE workspace_invitations.id = $1;

-- ===================================
-- WORKSPACE REQUESTS (Two-Key Control)
-- ===================================

-- name: CreateWorkspaceRequest :one
INSERT INTO workspace_requests (
    workspace_id, type, status, requested_by, reason, 
    target_user_id, new_parent_id, expires_at, sole_approver
)
VALUES ($1, $2, 'pending', $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetWorkspaceRequestByID :one
SELECT * FROM workspace_requests
WHERE workspace_requests.id = $1;

-- name: ListPendingRequestsForWorkspace :many
SELECT 
    requests.*, 
    requester_user.email AS requester_email, 
    requester_profile.display_name AS requester_name
FROM workspace_requests AS requests
JOIN users AS requester_user 
    ON requester_user.id = requests.requested_by
LEFT JOIN profiles AS requester_profile 
    ON requester_profile.user_id = requests.requested_by
WHERE requests.workspace_id = $1 
  AND requests.status = 'pending' 
  AND requests.expires_at > now()
ORDER BY requests.created_at DESC;

-- name: ListPendingRequestsForApprover :many
-- Lists pending requests for children where the user has oversight on the immediate parent
SELECT 
    requests.*, 
    workspaces.name AS workspace_name, 
    requester_user.email AS requester_email, 
    requester_profile.display_name AS requester_name
FROM workspace_requests AS requests
JOIN workspaces 
    ON workspaces.id = requests.workspace_id
JOIN users AS requester_user 
    ON requester_user.id = requests.requested_by
LEFT JOIN profiles AS requester_profile 
    ON requester_profile.user_id = requests.requested_by
JOIN workspace_closure AS closure 
    ON closure.descendant_id = requests.workspace_id 
   AND closure.depth = 1
JOIN workspace_members AS parent_member 
    ON parent_member.workspace_id = closure.ancestor_id 
   AND parent_member.user_id = $1 
   AND parent_member.status = 'active'
WHERE requests.status = 'pending' 
  AND requests.expires_at > now()
  AND (requests.requested_by != $1 OR requests.sole_approver = true)
ORDER BY requests.created_at DESC;

-- name: DecideWorkspaceRequest :one
UPDATE workspace_requests
SET status = $2,
    decided_by = $3,
    decided_at = now()
WHERE workspace_requests.id = $1 
  AND workspace_requests.status = 'pending' 
  AND workspace_requests.expires_at > now()
RETURNING *;

-- name: CancelWorkspaceRequest :one
UPDATE workspace_requests
SET status = 'cancelled'
WHERE workspace_requests.id = $1 
  AND workspace_requests.requested_by = $2 
  AND workspace_requests.status = 'pending'
RETURNING *;

-- ===================================
-- WORKSPACE AUDIT LOG (Immutable)
-- ===================================

-- name: CreateAuditLog :one
INSERT INTO workspace_audit (workspace_id, actor_id, action, target_type, target_id, metadata)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListAuditLogsForWorkspace :many
SELECT 
    audit.*, 
    actor_user.email AS actor_email, 
    actor_profile.display_name AS actor_name
FROM workspace_audit AS audit
LEFT JOIN users AS actor_user 
    ON actor_user.id = audit.actor_id
LEFT JOIN profiles AS actor_profile 
    ON actor_profile.user_id = audit.actor_id
WHERE audit.workspace_id = $1
ORDER BY audit.created_at DESC
LIMIT $2 OFFSET $3;
