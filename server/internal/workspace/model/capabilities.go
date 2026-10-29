package model

// Capability constants as defined in the Olona Governance Spec (v1).
const (
	CapWorkspaceView              = "workspace.view"
	CapWorkspaceManageMembers     = "workspace.manage_members"
	CapWorkspaceManageSettings    = "workspace.manage_settings"
	CapWorkspaceCreateChild       = "workspace.create_child"
	CapWorkspaceViewAudit         = "workspace.view_audit"
	CapWorkspaceRequestArchive    = "workspace.request_archive"
	CapWorkspaceRequestDelete     = "workspace.request_delete"
	CapWorkspaceRequestTransfer   = "workspace.request_transfer"
	CapWorkspaceRequestMove       = "workspace.request_move"
	CapWorkspaceOverseeChild      = "workspace.oversee_child"
	CapWorkspaceTransferOwnership = "workspace.transfer_ownership"
)

// AllCapabilities contains every standard capability.
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
var (
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

	MemberCapabilities = []string{
		CapWorkspaceView,
	}

	ViewerCapabilities = []string{
		CapWorkspaceView,
	}

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
