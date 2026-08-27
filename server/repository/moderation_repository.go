package repository

import (
	"github.com/ikafly144/au_mod_installer/server/model"
)

type ModerationRepository interface {
	CreateAuditLog(log *model.ModAuditLog) error
	GetAuditLogs(modID string, limit int) ([]model.ModAuditLog, error)

	CreateReport(report *model.ModReport) (string, error)
	GetReport(id string) (*model.ModReport, error)
	GetPendingReports() ([]model.ModReport, error)
	GetReportsByModID(modID string) ([]model.ModReport, error)
	UpdateReport(report *model.ModReport) error
}
