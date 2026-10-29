// Package repository provides the persistence layer for the workspace domain.
// It wraps the sqlc-generated *db.Queries and translates db-level errors
// into domain-safe sentinel errors. Higher layers (service/handler) should never
// import the db package directly.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "server/internal/platform/db/generated"
)

// Sentinel errors exposed by the workspace repository.
var (
	ErrNotFound         = errors.New("not found")
	ErrAlreadyExists    = errors.New("record already exists")
	ErrPermissionDenied = errors.New("insufficient database privilege")
	ErrForeignKey       = errors.New("referenced record does not exist")
	ErrInvalidState     = errors.New("invalid state transition")
)

// TxBeginner opens a database transaction. *pgxpool.Pool satisfies it.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// WorkspaceRepository defines the persistence contract for all workspace operations.
type WorkspaceRepository interface {
	// ===================================
	// WORKSPACES
	// ===================================
	CreateRootWorkspace(ctx context.Context, id uuid.UUID, name string, description *string, createdBy uuid.UUID) (db.Workspace, error)
	CreateChildWorkspace(ctx context.Context, id, parentID, rootID uuid.UUID, name string, description *string, singlePrincipal bool, createdBy uuid.UUID) (db.Workspace, error)
	GetWorkspaceByID(ctx context.Context, id uuid.UUID) (db.Workspace, error)
	UpdateWorkspaceName(ctx context.Context, id uuid.UUID, name string, description *string) (db.Workspace, error)
	SuspendWorkspace(ctx context.Context, id uuid.UUID, suspendedBy *uuid.UUID) (db.Workspace, error)
	ResumeWorkspace(ctx context.Context, id uuid.UUID) (db.Workspace, error)
	SetWorkspaceStatus(ctx context.Context, id uuid.UUID, status db.WorkspaceStatus) (db.Workspace, error)
	CountActiveChildren(ctx context.Context, parentID *uuid.UUID) (int64, error)
	MoveWorkspace(ctx context.Context, id uuid.UUID, newParentID *uuid.UUID) (db.Workspace, error)

	// ===================================
	// WORKSPACE CLOSURE
	// ===================================
	InsertRootClosure(ctx context.Context, rootID uuid.UUID) error
	InsertChildClosure(ctx context.Context, childID, parentID uuid.UUID) error
	IsAncestor(ctx context.Context, ancestorID, descendantID uuid.UUID) (bool, error)
	GetAncestorAtDepth(ctx context.Context, descendantID uuid.UUID, depth int32) (uuid.UUID, error)
	GetImmediateParentClosure(ctx context.Context, descendantID uuid.UUID) (uuid.UUID, error)
	DeleteSubtreeClosureForMove(ctx context.Context, workspaceID uuid.UUID) error
	RebuildClosureForMovedSubtree(ctx context.Context, workspaceID, newParentID uuid.UUID) error

	// ===================================
	// WORKSPACE MEMBERS & RELOCATION
	// ===================================
	AddWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, tier db.WorkspaceTier, capabilities []string, invitedBy *uuid.UUID) (db.WorkspaceMember, error)
	GetActiveMember(ctx context.Context, workspaceID, userID uuid.UUID) (db.WorkspaceMember, error)
	GetActiveOwner(ctx context.Context, workspaceID uuid.UUID) (db.WorkspaceMember, error)
	ListActiveMembers(ctx context.Context, workspaceID uuid.UUID) ([]db.ListActiveMembersRow, error)
	UpdateMemberTierAndCapabilities(ctx context.Context, workspaceID, userID uuid.UUID, tier db.WorkspaceTier, capabilities []string) (db.WorkspaceMember, error)
	RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error
	TransferOwnershipInWorkspace(ctx context.Context, workspaceID, currentOwnerID, targetAdminID uuid.UUID) error

	// ===================================
	// VISIBILITY & EFFECTIVE ACCESS
	// ===================================
	ListVisibleWorkspacesForUser(ctx context.Context, userID uuid.UUID) ([]db.ListVisibleWorkspacesForUserRow, error)
	GetEffectiveAccess(ctx context.Context, userID, workspaceID uuid.UUID) (db.GetEffectiveAccessRow, error)

	// ===================================
	// WORKSPACE INVITATIONS
	// ===================================
	CreateWorkspaceInvitation(ctx context.Context, workspaceID uuid.UUID, email string, tier db.WorkspaceTier, capabilities []string, tokenHash string, invitedBy uuid.UUID, expiresAt time.Time) (db.WorkspaceInvitation, error)
	GetWorkspaceInvitationByToken(ctx context.Context, tokenHash string) (db.WorkspaceInvitation, error)
	ListPendingInvitations(ctx context.Context, workspaceID uuid.UUID) ([]db.WorkspaceInvitation, error)
	MarkWorkspaceInvitationAccepted(ctx context.Context, id uuid.UUID) error
	RevokeWorkspaceInvitation(ctx context.Context, id uuid.UUID) error

	// ===================================
	// WORKSPACE REQUESTS (Two-Key Control)
	// ===================================
	CreateWorkspaceRequest(ctx context.Context, workspaceID uuid.UUID, reqType db.WorkspaceRequestType, requestedBy uuid.UUID, reason *string, targetUserID, newParentID *uuid.UUID, expiresAt time.Time, soleApprover bool) (db.WorkspaceRequest, error)
	GetWorkspaceRequestByID(ctx context.Context, id uuid.UUID) (db.WorkspaceRequest, error)
	ListPendingRequestsForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]db.ListPendingRequestsForWorkspaceRow, error)
	ListPendingRequestsForApprover(ctx context.Context, userID uuid.UUID) ([]db.ListPendingRequestsForApproverRow, error)
	DecideWorkspaceRequest(ctx context.Context, id uuid.UUID, status db.WorkspaceRequestStatus, decidedBy *uuid.UUID) (db.WorkspaceRequest, error)
	CancelWorkspaceRequest(ctx context.Context, id, requestedBy uuid.UUID) (db.WorkspaceRequest, error)

	// ===================================
	// WORKSPACE AUDIT LOG
	// ===================================
	CreateAuditLog(ctx context.Context, workspaceID uuid.UUID, actorID *uuid.UUID, action, targetType string, targetID *uuid.UUID, metadata map[string]interface{}) (db.WorkspaceAudit, error)
	ListAuditLogsForWorkspace(ctx context.Context, workspaceID uuid.UUID, limit, offset int32) ([]db.ListAuditLogsForWorkspaceRow, error)

	// ===================================
	// TRANSACTIONS
	// ===================================
	WithTx(ctx context.Context, fn func(WorkspaceRepository) error) error
}

// ErrTransactionsUnsupported is returned by WithTx when the repository was
// built without a transaction source.
var ErrTransactionsUnsupported = errors.New(
	"repository: transactions are not configured; pass WithTxSource when constructing the repository",
)

// RepositoryOption customises a repository at construction time.
type RepositoryOption func(*workspaceRepositoryImpl)

// WithTxSource enables WithTx on the repository.
func WithTxSource(beginner TxBeginner) RepositoryOption {
	return func(repo *workspaceRepositoryImpl) {
		repo.beginner = beginner
	}
}

type workspaceRepositoryImpl struct {
	queries  db.Querier
	beginner TxBeginner
}

// NewWorkspaceRepository creates a new WorkspaceRepository instance backed by db.Querier.
func NewWorkspaceRepository(queries db.Querier, opts ...RepositoryOption) WorkspaceRepository {
	repo := &workspaceRepositoryImpl{queries: queries}
	for _, opt := range opts {
		opt(repo)
	}
	return repo
}

// WithTx runs fn inside a single database transaction.
func (repo *workspaceRepositoryImpl) WithTx(ctx context.Context, fn func(WorkspaceRepository) error) error {
	if repo.beginner == nil {
		return ErrTransactionsUnsupported
	}

	tx, err := repo.beginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("workspace repository: begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				log.Printf("workspace repository: rollback failed: %v", rbErr)
			}
		}
	}()

	qtx := db.New(tx)
	txRepo := &workspaceRepositoryImpl{
		queries:  qtx,
		beginner: repo.beginner,
	}

	if err := fn(txRepo); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("workspace repository: commit transaction: %w", err)
	}
	committed = true
	return nil
}

// ===================================
// IMPLEMENTATIONS: WORKSPACES
// ===================================

func (repo *workspaceRepositoryImpl) CreateRootWorkspace(ctx context.Context, id uuid.UUID, name string, description *string, createdBy uuid.UUID) (db.Workspace, error) {
	ws, err := repo.queries.CreateRootWorkspace(ctx, db.CreateRootWorkspaceParams{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedBy:   createdBy,
	})
	return ws, wrap(err)
}

func (repo *workspaceRepositoryImpl) CreateChildWorkspace(ctx context.Context, id, parentID, rootID uuid.UUID, name string, description *string, singlePrincipal bool, createdBy uuid.UUID) (db.Workspace, error) {
	ws, err := repo.queries.CreateChildWorkspace(ctx, db.CreateChildWorkspaceParams{
		ID:              id,
		ParentID:        toUUID(parentID),
		RootID:          rootID,
		Name:            name,
		Description:     description,
		SinglePrincipal: singlePrincipal,
		CreatedBy:       createdBy,
	})
	return ws, wrap(err)
}

func (repo *workspaceRepositoryImpl) GetWorkspaceByID(ctx context.Context, id uuid.UUID) (db.Workspace, error) {
	ws, err := repo.queries.GetWorkspaceByID(ctx, id)
	return ws, wrap(err)
}

func (repo *workspaceRepositoryImpl) UpdateWorkspaceName(ctx context.Context, id uuid.UUID, name string, description *string) (db.Workspace, error) {
	ws, err := repo.queries.UpdateWorkspaceName(ctx, db.UpdateWorkspaceNameParams{
		ID:          id,
		Name:        name,
		Description: description,
	})
	return ws, wrap(err)
}

func (repo *workspaceRepositoryImpl) SuspendWorkspace(ctx context.Context, id uuid.UUID, suspendedBy *uuid.UUID) (db.Workspace, error) {
	ws, err := repo.queries.SuspendWorkspace(ctx, db.SuspendWorkspaceParams{
		ID:          id,
		SuspendedBy: toNullableUUID(suspendedBy),
	})
	return ws, wrap(err)
}

func (repo *workspaceRepositoryImpl) ResumeWorkspace(ctx context.Context, id uuid.UUID) (db.Workspace, error) {
	ws, err := repo.queries.ResumeWorkspace(ctx, id)
	return ws, wrap(err)
}

func (repo *workspaceRepositoryImpl) SetWorkspaceStatus(ctx context.Context, id uuid.UUID, status db.WorkspaceStatus) (db.Workspace, error) {
	ws, err := repo.queries.SetWorkspaceStatus(ctx, db.SetWorkspaceStatusParams{
		ID:     id,
		Status: status,
	})
	return ws, wrap(err)
}

func (repo *workspaceRepositoryImpl) CountActiveChildren(ctx context.Context, parentID *uuid.UUID) (int64, error) {
	count, err := repo.queries.CountActiveChildren(ctx, toNullableUUID(parentID))
	return count, wrap(err)
}

func (repo *workspaceRepositoryImpl) MoveWorkspace(ctx context.Context, id uuid.UUID, newParentID *uuid.UUID) (db.Workspace, error) {
	ws, err := repo.queries.MoveWorkspace(ctx, db.MoveWorkspaceParams{
		ID:       id,
		ParentID: toNullableUUID(newParentID),
	})
	return ws, wrap(err)
}

// ===================================
// IMPLEMENTATIONS: CLOSURE
// ===================================

func (repo *workspaceRepositoryImpl) InsertRootClosure(ctx context.Context, rootID uuid.UUID) error {
	err := repo.queries.InsertRootClosure(ctx, rootID)
	return wrap(err)
}

func (repo *workspaceRepositoryImpl) InsertChildClosure(ctx context.Context, childID, parentID uuid.UUID) error {
	err := repo.queries.InsertChildClosure(ctx, db.InsertChildClosureParams{
		Column1:      childID,
		DescendantID: parentID,
	})
	return wrap(err)
}

func (repo *workspaceRepositoryImpl) IsAncestor(ctx context.Context, ancestorID, descendantID uuid.UUID) (bool, error) {
	ok, err := repo.queries.IsAncestor(ctx, db.IsAncestorParams{
		AncestorID:   ancestorID,
		DescendantID: descendantID,
	})
	return ok, wrap(err)
}

func (repo *workspaceRepositoryImpl) GetAncestorAtDepth(ctx context.Context, descendantID uuid.UUID, depth int32) (uuid.UUID, error) {
	id, err := repo.queries.GetAncestorAtDepth(ctx, db.GetAncestorAtDepthParams{
		DescendantID: descendantID,
		Depth:        depth,
	})
	return id, wrap(err)
}

func (repo *workspaceRepositoryImpl) GetImmediateParentClosure(ctx context.Context, descendantID uuid.UUID) (uuid.UUID, error) {
	id, err := repo.queries.GetImmediateParentClosure(ctx, descendantID)
	return id, wrap(err)
}

func (repo *workspaceRepositoryImpl) DeleteSubtreeClosureForMove(ctx context.Context, workspaceID uuid.UUID) error {
	err := repo.queries.DeleteSubtreeClosureForMove(ctx, workspaceID)
	return wrap(err)
}

func (repo *workspaceRepositoryImpl) RebuildClosureForMovedSubtree(ctx context.Context, workspaceID, newParentID uuid.UUID) error {
	err := repo.queries.RebuildClosureForMovedSubtree(ctx, db.RebuildClosureForMovedSubtreeParams{
		AncestorID:   workspaceID,
		DescendantID: newParentID,
	})
	return wrap(err)
}

// ===================================
// IMPLEMENTATIONS: MEMBERS & RELOCATION
// ===================================

func (repo *workspaceRepositoryImpl) AddWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID, tier db.WorkspaceTier, capabilities []string, invitedBy *uuid.UUID) (db.WorkspaceMember, error) {
	capsJSON, err := json.Marshal(capabilities)
	if err != nil {
		return db.WorkspaceMember{}, fmt.Errorf("marshal capabilities: %w", err)
	}

	member, err := repo.queries.AddWorkspaceMember(ctx, db.AddWorkspaceMemberParams{
		WorkspaceID:  workspaceID,
		UserID:       userID,
		Tier:         tier,
		Capabilities: capsJSON,
		InvitedBy:    toNullableUUID(invitedBy),
	})
	return member, wrap(err)
}

func (repo *workspaceRepositoryImpl) GetActiveMember(ctx context.Context, workspaceID, userID uuid.UUID) (db.WorkspaceMember, error) {
	m, err := repo.queries.GetActiveMember(ctx, db.GetActiveMemberParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
	return m, wrap(err)
}

func (repo *workspaceRepositoryImpl) GetActiveOwner(ctx context.Context, workspaceID uuid.UUID) (db.WorkspaceMember, error) {
	m, err := repo.queries.GetActiveOwner(ctx, workspaceID)
	return m, wrap(err)
}

func (repo *workspaceRepositoryImpl) ListActiveMembers(ctx context.Context, workspaceID uuid.UUID) ([]db.ListActiveMembersRow, error) {
	members, err := repo.queries.ListActiveMembers(ctx, workspaceID)
	return members, wrap(err)
}

func (repo *workspaceRepositoryImpl) UpdateMemberTierAndCapabilities(ctx context.Context, workspaceID, userID uuid.UUID, tier db.WorkspaceTier, capabilities []string) (db.WorkspaceMember, error) {
	capsJSON, err := json.Marshal(capabilities)
	if err != nil {
		return db.WorkspaceMember{}, fmt.Errorf("marshal capabilities: %w", err)
	}

	m, err := repo.queries.UpdateMemberTierAndCapabilities(ctx, db.UpdateMemberTierAndCapabilitiesParams{
		WorkspaceID:  workspaceID,
		UserID:       userID,
		Tier:         tier,
		Capabilities: capsJSON,
	})
	return m, wrap(err)
}

func (repo *workspaceRepositoryImpl) RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error {
	err := repo.queries.RemoveMember(ctx, db.RemoveMemberParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
	return wrap(err)
}

func (repo *workspaceRepositoryImpl) TransferOwnershipInWorkspace(ctx context.Context, workspaceID, currentOwnerID, targetAdminID uuid.UUID) error {
	err := repo.queries.TransferOwnershipInWorkspace(ctx, db.TransferOwnershipInWorkspaceParams{
		WorkspaceID: workspaceID,
		UserID:      currentOwnerID,
		UserID_2:    targetAdminID,
	})
	return wrap(err)
}

// ===================================
// IMPLEMENTATIONS: VISIBILITY & ACCESS
// ===================================

func (repo *workspaceRepositoryImpl) ListVisibleWorkspacesForUser(ctx context.Context, userID uuid.UUID) ([]db.ListVisibleWorkspacesForUserRow, error) {
	rows, err := repo.queries.ListVisibleWorkspacesForUser(ctx, userID)
	return rows, wrap(err)
}

func (repo *workspaceRepositoryImpl) GetEffectiveAccess(ctx context.Context, userID, workspaceID uuid.UUID) (db.GetEffectiveAccessRow, error) {
	row, err := repo.queries.GetEffectiveAccess(ctx, db.GetEffectiveAccessParams{
		UserID:       userID,
		DescendantID: workspaceID,
	})
	return row, wrap(err)
}

// ===================================
// IMPLEMENTATIONS: INVITATIONS
// ===================================

func (repo *workspaceRepositoryImpl) CreateWorkspaceInvitation(ctx context.Context, workspaceID uuid.UUID, email string, tier db.WorkspaceTier, capabilities []string, tokenHash string, invitedBy uuid.UUID, expiresAt time.Time) (db.WorkspaceInvitation, error) {
	capsJSON, err := json.Marshal(capabilities)
	if err != nil {
		return db.WorkspaceInvitation{}, fmt.Errorf("marshal capabilities: %w", err)
	}

	inv, err := repo.queries.CreateWorkspaceInvitation(ctx, db.CreateWorkspaceInvitationParams{
		WorkspaceID:  workspaceID,
		Email:        email,
		Tier:         tier,
		Capabilities: capsJSON,
		TokenHash:    tokenHash,
		InvitedBy:    invitedBy,
		ExpiresAt:    expiresAt,
	})
	return inv, wrap(err)
}

func (repo *workspaceRepositoryImpl) GetWorkspaceInvitationByToken(ctx context.Context, tokenHash string) (db.WorkspaceInvitation, error) {
	inv, err := repo.queries.GetWorkspaceInvitationByToken(ctx, tokenHash)
	return inv, wrap(err)
}

func (repo *workspaceRepositoryImpl) ListPendingInvitations(ctx context.Context, workspaceID uuid.UUID) ([]db.WorkspaceInvitation, error) {
	invs, err := repo.queries.ListPendingInvitations(ctx, workspaceID)
	return invs, wrap(err)
}

func (repo *workspaceRepositoryImpl) MarkWorkspaceInvitationAccepted(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.MarkWorkspaceInvitationAccepted(ctx, id)
	return wrap(err)
}

func (repo *workspaceRepositoryImpl) RevokeWorkspaceInvitation(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.RevokeWorkspaceInvitation(ctx, id)
	return wrap(err)
}

// ===================================
// IMPLEMENTATIONS: REQUESTS
// ===================================

func (repo *workspaceRepositoryImpl) CreateWorkspaceRequest(ctx context.Context, workspaceID uuid.UUID, reqType db.WorkspaceRequestType, requestedBy uuid.UUID, reason *string, targetUserID, newParentID *uuid.UUID, expiresAt time.Time, soleApprover bool) (db.WorkspaceRequest, error) {
	req, err := repo.queries.CreateWorkspaceRequest(ctx, db.CreateWorkspaceRequestParams{
		WorkspaceID:  workspaceID,
		Type:         reqType,
		RequestedBy:  requestedBy,
		Reason:       reason,
		TargetUserID: toNullableUUID(targetUserID),
		NewParentID:  toNullableUUID(newParentID),
		ExpiresAt:    expiresAt,
		SoleApprover: soleApprover,
	})
	return req, wrap(err)
}

func (repo *workspaceRepositoryImpl) GetWorkspaceRequestByID(ctx context.Context, id uuid.UUID) (db.WorkspaceRequest, error) {
	req, err := repo.queries.GetWorkspaceRequestByID(ctx, id)
	return req, wrap(err)
}

func (repo *workspaceRepositoryImpl) ListPendingRequestsForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]db.ListPendingRequestsForWorkspaceRow, error) {
	reqs, err := repo.queries.ListPendingRequestsForWorkspace(ctx, workspaceID)
	return reqs, wrap(err)
}

func (repo *workspaceRepositoryImpl) ListPendingRequestsForApprover(ctx context.Context, userID uuid.UUID) ([]db.ListPendingRequestsForApproverRow, error) {
	reqs, err := repo.queries.ListPendingRequestsForApprover(ctx, userID)
	return reqs, wrap(err)
}

func (repo *workspaceRepositoryImpl) DecideWorkspaceRequest(ctx context.Context, id uuid.UUID, status db.WorkspaceRequestStatus, decidedBy *uuid.UUID) (db.WorkspaceRequest, error) {
	req, err := repo.queries.DecideWorkspaceRequest(ctx, db.DecideWorkspaceRequestParams{
		ID:        id,
		Status:    status,
		DecidedBy: toNullableUUID(decidedBy),
	})
	return req, wrap(err)
}

func (repo *workspaceRepositoryImpl) CancelWorkspaceRequest(ctx context.Context, id, requestedBy uuid.UUID) (db.WorkspaceRequest, error) {
	req, err := repo.queries.CancelWorkspaceRequest(ctx, db.CancelWorkspaceRequestParams{
		ID:          id,
		RequestedBy: requestedBy,
	})
	return req, wrap(err)
}

// ===================================
// IMPLEMENTATIONS: AUDIT LOG
// ===================================

func (repo *workspaceRepositoryImpl) CreateAuditLog(ctx context.Context, workspaceID uuid.UUID, actorID *uuid.UUID, action, targetType string, targetID *uuid.UUID, metadata map[string]interface{}) (db.WorkspaceAudit, error) {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	audit, err := repo.queries.CreateAuditLog(ctx, db.CreateAuditLogParams{
		WorkspaceID: workspaceID,
		ActorID:     toNullableUUID(actorID),
		Action:      action,
		TargetType:  targetType,
		TargetID:    toNullableUUID(targetID),
		Metadata:    metaJSON,
	})
	return audit, wrap(err)
}

func (repo *workspaceRepositoryImpl) ListAuditLogsForWorkspace(ctx context.Context, workspaceID uuid.UUID, limit, offset int32) ([]db.ListAuditLogsForWorkspaceRow, error) {
	audits, err := repo.queries.ListAuditLogsForWorkspace(ctx, db.ListAuditLogsForWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       limit,
		Offset:      offset,
	})
	return audits, wrap(err)
}

// ===================================
// HELPERS & ERROR HANDLING
// ===================================

func toUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toNullableUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

// wrap maps low-level pgx/postgres errors into domain sentinel errors.
func wrap(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42501": // insufficient_privilege
			return ErrPermissionDenied
		case "23505": // unique_violation
			return ErrAlreadyExists
		case "23503": // foreign_key_violation
			return ErrForeignKey
		case "23514": // check_violation
			return ErrInvalidState
		}
	}

	return err
}
