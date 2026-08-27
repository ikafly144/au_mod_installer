package model

import (
	"time"
)

type SubmissionStatus string

const (
	SubmissionStatusDraft         SubmissionStatus = "draft"
	SubmissionStatusPendingReview SubmissionStatus = "pending_review"
	SubmissionStatusScanning      SubmissionStatus = "scanning"
	SubmissionStatusApproved      SubmissionStatus = "approved"
	SubmissionStatusRejected      SubmissionStatus = "rejected"
)

// ModSubmission represents a submission for creating or modifying a Mod entry before approval.
type ModSubmission struct {
	ID             string           `gorm:"primaryKey" json:"id"`
	ModID          string           `gorm:"index:idx_mod_sub_mod_id;not null" json:"mod_id"`
	Name           string           `gorm:"not null" json:"name"`
	Description    string           `gorm:"not null" json:"description"`
	AuthorName     string           `gorm:"not null" json:"author_name"`
	SubmitterID    string           `gorm:"index:idx_mod_sub_submitter;not null" json:"submitter_id"` // Discord User ID
	ThumbnailURL   string           `json:"thumbnail_url,omitempty"`
	Status         SubmissionStatus `gorm:"index:idx_mod_sub_status;not null;default:'pending_review'" json:"status"`
	ReviewThreadID string           `json:"review_thread_id,omitempty"`
	RejectionReason string          `json:"rejection_reason,omitempty"`
	ReviewedBy     string           `json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time       `json:"reviewed_at,omitempty"`
	CreatedAt      time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

// VersionSubmission represents a submission for a new version of a Mod before approval.
type VersionSubmission struct {
	ID          string           `gorm:"primaryKey" json:"id"`
	ModID       string           `gorm:"index:idx_ver_sub_mod_id;not null" json:"mod_id"`
	VersionID   string           `gorm:"not null" json:"version_id"`
	Changelog   string           `json:"changelog,omitempty"`
	SubmitterID string           `gorm:"index:idx_ver_sub_submitter;not null" json:"submitter_id"`
	Status      SubmissionStatus `gorm:"index:idx_ver_sub_status;not null;default:'pending_review'" json:"status"`

	// Target and File metadata
	TargetPlatform TargetPlatform `gorm:"not null;default:'any'" json:"target_platform"`
	ContentType    FileType       `gorm:"not null;default:'archive'" json:"content_type"`
	Filename       string         `gorm:"not null" json:"filename"`
	FileSize       int64          `gorm:"not null" json:"file_size"`
	ExtractPath    *string        `gorm:"default:null" json:"extract_path,omitempty"`
	Hashes         StringMap      `gorm:"type:json" json:"hashes"`
	Downloads      StringArray    `gorm:"type:json" json:"downloads"`
	Dependencies   DependencyArray `gorm:"type:json" json:"dependencies"`
	Features       Features       `gorm:"type:json" json:"features"`

	// Security & Scan information
	VirusTotalStatus string `gorm:"default:'unscanned'" json:"vt_status"` // clean, suspicious, malicious, unscanned, error
	VirusTotalScore  int    `gorm:"default:0" json:"vt_score"`             // positive detections count
	VirusTotalURL    string `json:"vt_url,omitempty"`

	// Review details
	ReviewThreadID  string     `json:"review_thread_id,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
	ReviewedBy      string     `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
