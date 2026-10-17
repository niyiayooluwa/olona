package model

// Role is the account type
type Role string

const (
	// RoleAdmin is the account owner.
	RoleAdmin Role = "admin"
	// RoleStaff is a member of the team.
	RoleStaff Role = "staff"
	// RoleClient is someone who pays.
	RoleClient Role = "client"
)
