package repository

import (
	"github.com/ikafly144/au_mod_installer/server/model"
)

type SubmissionRepository interface {
	CreateModSubmission(sub *model.ModSubmission) (string, error)
	GetModSubmission(id string) (*model.ModSubmission, error)
	GetModSubmissionsByModID(modID string) ([]model.ModSubmission, error)
	GetPendingModSubmissions() ([]model.ModSubmission, error)
	GetUserModSubmissions(submitterID string) ([]model.ModSubmission, error)
	UpdateModSubmission(sub *model.ModSubmission) error
	DeleteModSubmission(id string) error

	CreateVersionSubmission(sub *model.VersionSubmission) (string, error)
	GetVersionSubmission(id string) (*model.VersionSubmission, error)
	GetPendingVersionSubmissions(modID string) ([]model.VersionSubmission, error)
	GetUserVersionSubmissions(submitterID string) ([]model.VersionSubmission, error)
	UpdateVersionSubmission(sub *model.VersionSubmission) error
	DeleteVersionSubmission(id string) error
}
