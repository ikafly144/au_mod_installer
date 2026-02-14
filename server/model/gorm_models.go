package model

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// User represents the users table
type User struct {
	ID          int            `gorm:"primaryKey;autoIncrement"`
	DiscordID   string         `gorm:"unique;not null"`
	Username    string         `gorm:"not null"`
	DisplayName string         `gorm:"default:null"`
	AvatarURL   string         `gorm:"default:null"`
	IsAdmin     bool           `gorm:"default:false"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// Mod represents the mods table
type Mod struct {
	ID              string         `gorm:"primaryKey"`
	Name            string         `gorm:"not null"`
	Description     string         `gorm:"default:null"`
	AuthorID        *int           `gorm:"default:null"` // Foreign Key to User
	AuthorName      string         `gorm:"default:null"` // Fallback name
	Type            string         `gorm:"default:null"`
	ThumbnailURL    string         `gorm:"default:null"`
	WebsiteURL      string         `gorm:"default:null"`
	GitHubRepo      string         `gorm:"default:null"`
	LatestVersionID string         `gorm:"default:null"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Relationships
	Author   *User        `gorm:"foreignKey:AuthorID"`
	Versions []ModVersion `gorm:"foreignKey:ModID;constraint:OnDelete:CASCADE"`
}

// ModVersion represents the mod_versions table
type ModVersion struct {
	ModID     string         `gorm:"primaryKey"`
	VersionID string         `gorm:"primaryKey"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relationships
	Files        []ModFile               `gorm:"foreignKey:ModID,VersionID;constraint:OnDelete:CASCADE"`
	Dependencies []ModDependency         `gorm:"foreignKey:ModID,VersionID;constraint:OnDelete:CASCADE"`
	GameVersions []ModVersionGameVersion `gorm:"foreignKey:ModID,VersionID;constraint:OnDelete:CASCADE"`
}

// ModFile represents the mod_files table
type ModFile struct {
	ID        int    `gorm:"primaryKey;autoIncrement"`
	ModID     string `gorm:"index"`
	VersionID string `gorm:"index"`
	FileType  string `gorm:"default:null"`
	Path      string `gorm:"default:null"`
	URL       string `gorm:"default:null"`
	// Use lib/pq for array support in Postgres
	CompatibleBinaryTypes pq.StringArray `gorm:"type:text[]"`
}

// ModDependency represents the mod_dependencies table
type ModDependency struct {
	ID                int    `gorm:"primaryKey;autoIncrement"`
	ModID             string `gorm:"index"`
	VersionID         string `gorm:"index"`
	DependencyID      string
	DependencyVersion string
	DependencyType    string
}

// ModVersionGameVersion represents the mod_version_game_versions table
type ModVersionGameVersion struct {
	ModID       string `gorm:"primaryKey"`
	VersionID   string `gorm:"primaryKey"`
	GameVersion string `gorm:"primaryKey"`
}

// Helper to convert GORM Model to pkg/modmgr/model types is needed in Repository implementation
