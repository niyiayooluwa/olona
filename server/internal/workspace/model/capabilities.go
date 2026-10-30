package model

// Capability constants as defined in the Olona Governance Spec (v1 §4).
// Every action in the API and UI gates on these explicit capability strings
// rather than hardcoded role names.
const (
	// CapWorkspaceView allows viewing the workspace, its metadata, its dashboard,
	// and listing its visible members.
	CapWorkspaceView = "workspace.view"

	// CapWorkspaceManageMembers allows inviting new members (admin, member, viewer),
	// changing non-owner member roles/capabilities, and removing non-owner members.
	CapWorkspaceManageMembers = "workspace.manage_members"

	// CapWorkspaceManageSettings allows updating non-charter workspace settings,
	// such as renaming the workspace and modifying its description.
	CapWorkspaceManageSettings = "workspace.manage_settings"

	// CapWorkspaceCreateChild allows attaching and initializing a new subsidiary
	// child workspace directly under this workspace, including promoting one of
	// this workspace's admins to become the child owner (atomic relocation).
	CapWorkspaceCreateChild = "workspace.create_child"

	// CapWorkspaceViewAudit allows reading the workspace's immutable append-only
	// audit trail log to inspect governance actions, timestamps, and actors.
	CapWorkspaceViewAudit = "workspace.view_audit"

	// CapWorkspaceRequestArchive allows opening a formal Two-Key Control request
	// to archive this workspace. The request is submitted to the parent workspace's
	// approval inbox and requires parent oversight approval to execute.
	CapWorkspaceRequestArchive = "workspace.request_archive"

	// CapWorkspaceRequestDelete allows opening a formal Two-Key Control request
	// to schedule this workspace for permanent deletion after the 30-day grace period.
	// Requires parent oversight approval.
	CapWorkspaceRequestDelete = "workspace.request_delete"

	// CapWorkspaceRequestTransfer allows opening a formal Two-Key Control request
	// to transfer the workspace's sole ownership to an existing active admin of this workspace.
	// Requires parent oversight approval.
	CapWorkspaceRequestTransfer = "workspace.request_transfer"

	// CapWorkspaceRequestMove allows opening a formal Two-Key Control request
	// to reparent/move this entire workspace branch under a different parent workspace
	// within the same organization tree.
	CapWorkspaceRequestMove = "workspace.request_move"

	// CapWorkspaceOverseeChild is an OVERSIGHT capability held by an ancestor over
	// its descendants. It allows inspecting child metadata, viewing child audit logs,
	// immediately suspending or resuming a child branch without approval, and
	// approving or rejecting sensitive Two-Key Control requests opened by the child.
	// It does NOT grant daily operational privileges inside the child.
	CapWorkspaceOverseeChild = "workspace.oversee_child"

	// CapWorkspaceTransferOwnership allows immediate intra-workspace ownership
	// handover where authorized by governing policy.
	CapWorkspaceTransferOwnership = "workspace.transfer_ownership"
)

// AllCapabilities contains every standard capability defined in v1.
var AllCapabilities = []string{
	CapWorkspaceView,
	CapWorkspaceManageMembers,
	CapWorkspaceManageSettings,
	CapWorkspaceCreateChild,
	CapWorkspaceViewAudit,
	CapWorkspaceRequestArchive,
	CapWorkspaceRequestDelete,
	CapWorkspaceRequestTransfer,
	CapWorkspaceRequestMove,
	CapWorkspaceOverseeChild,
	CapWorkspaceTransferOwnership,
}

// Preset bundles for v1 tiers.
// A preset is the default bundle of capabilities granted to a user when assigned a tier.
var (
	// OwnerCapabilities grants full operational, administrative, and request powers within the workspace.
	OwnerCapabilities = []string{
		CapWorkspaceView,
		CapWorkspaceManageMembers,
		CapWorkspaceManageSettings,
		CapWorkspaceCreateChild,
		CapWorkspaceViewAudit,
		CapWorkspaceRequestArchive,
		CapWorkspaceRequestDelete,
		CapWorkspaceRequestTransfer,
		CapWorkspaceRequestMove,
		CapWorkspaceTransferOwnership,
	}

	// AdminCapabilities grants member management, settings management, child creation,
	// audit viewing, and the right to open requests. It cannot directly transfer ownership.
	AdminCapabilities = []string{
		CapWorkspaceView,
		CapWorkspaceManageMembers,
		CapWorkspaceManageSettings,
		CapWorkspaceCreateChild,
		CapWorkspaceViewAudit,
		CapWorkspaceRequestArchive,
		CapWorkspaceRequestDelete,
		CapWorkspaceRequestTransfer,
		CapWorkspaceRequestMove,
	}

	// MemberCapabilities represents standard daily operational access (view-only in v1 before financial modules).
	MemberCapabilities = []string{
		CapWorkspaceView,
	}

	// ViewerCapabilities represents external auditors or observers who have read-only access.
	ViewerCapabilities = []string{
		CapWorkspaceView,
	}

	// OversightCapabilities is automatically derived for owners and admins of an ancestor workspace.
	// They hold oversight over all descendant branches without holding direct seats in those branches.
	OversightCapabilities = []string{
		CapWorkspaceView,
		CapWorkspaceViewAudit,
		CapWorkspaceOverseeChild,
	}
)

// EffectiveAccess represents what the caller is allowed to do in a workspace.
// This is returned in every workspace API response according to §11 of the spec.
type EffectiveAccess struct {
	Tier         *string  `json:"tier"` // "owner", "admin", "member", "viewer", or nil if oversight only
	Via          string   `json:"via"`  // "membership" or "oversight"
	Capabilities []string `json:"capabilities"`
}
