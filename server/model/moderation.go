package model

import (
	"time"
)

type ModStatus string

const (
	ModStatusApproved    ModStatus = "approved"
	ModStatusUnpublished ModStatus = "unpublished"
	ModStatusBanned      ModStatus = "banned"
)

type ModAuditAction string

const (
	AuditActionModSubmitted       ModAuditAction = "MOD_SUBMITTED"
	AuditActionModApproved        ModAuditAction = "MOD_APPROVED"
	AuditActionModRejected        ModAuditAction = "MOD_REJECTED"
	AuditActionModBanned          ModAuditAction = "MOD_BANNED"
	AuditActionModUnbanned        ModAuditAction = "MOD_UNBANNED"
	AuditActionModUnpublished     ModAuditAction = "MOD_UNPUBLISHED"
	AuditActionModPublished       ModAuditAction = "MOD_PUBLISHED"
	AuditActionVersionSubmitted   ModAuditAction = "VERSION_SUBMITTED"
	AuditActionVersionApproved    ModAuditAction = "VERSION_APPROVED"
	AuditActionVersionRejected    ModAuditAction = "VERSION_REJECTED"
	AuditActionCollaboratorAdded  ModAuditAction = "COLLABORATOR_ADDED"
	AuditActionCollaboratorRemoved ModAuditAction = "COLLABORATOR_REMOVED"
	AuditActionOwnershipTransferred ModAuditAction = "OWNERSHIP_TRANSFERRED"
	AuditActionModReported        ModAuditAction = "MOD_REPORTED"
	AuditActionReportResolved     ModAuditAction = "REPORT_RESOLVED"
	AuditActionReportDismissed    ModAuditAction = "REPORT_DISMISSED"
)

type ModAuditLog struct {
	ID        string         `gorm:"primaryKey" json:"id"`
	ModID     string         `gorm:"index:idx_audit_mod_id;not null" json:"mod_id"`
	Action    ModAuditAction `gorm:"index:idx_audit_action;not null" json:"action"`
	ActorID   string         `gorm:"index:idx_audit_actor;not null" json:"actor_id"` // Discord User ID
	Reason    string         `json:"reason,omitempty"`
	Details   StringMap      `gorm:"type:json" json:"details,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime;index:idx_audit_created" json:"created_at"`
}

type ReportCategory string

const (
	ReportCategoryMalware   ReportCategory = "malware"
	ReportCategoryCrash     ReportCategory = "game_crash"
	ReportCategoryCopyright ReportCategory = "copyright"
	ReportCategoryNSFW      ReportCategory = "nsfw"
	ReportCategorySpam      ReportCategory = "spam"
	ReportCategoryOther     ReportCategory = "other"
)

type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

type ModReport struct {
	ID          string         `gorm:"primaryKey" json:"id"`
	ModID       string         `gorm:"index:idx_report_mod_id;not null" json:"mod_id"`
	ReporterID  string         `gorm:"index:idx_report_reporter;not null" json:"reporter_id"` // Discord User ID
	Category    ReportCategory `gorm:"not null" json:"category"`
	Reason      string         `gorm:"not null" json:"reason"`
	Status      ReportStatus   `gorm:"index:idx_report_status;not null;default:'pending'" json:"status"`
	ActionTaken string         `json:"action_taken,omitempty"`
	ResolvedBy  string         `json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}
