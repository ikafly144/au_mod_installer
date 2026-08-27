package gorm

import (
	"github.com/ikafly144/au_mod_installer/server/model"
)

func (r *GormRepository) CreateAuditLog(log *model.ModAuditLog) error {
	result := r.db.Create(log)
	return result.Error
}

func (r *GormRepository) GetAuditLogs(modID string, limit int) ([]model.ModAuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var logs []model.ModAuditLog
	query := r.db.Model(&model.ModAuditLog{})
	if modID != "" {
		query = query.Where("mod_id = ?", modID)
	}
	result := query.Order("created_at DESC").Limit(limit).Find(&logs)
	if result.Error != nil {
		return nil, result.Error
	}
	return logs, nil
}

func (r *GormRepository) CreateReport(report *model.ModReport) (string, error) {
	result := r.db.Create(report)
	if result.Error != nil {
		return "", result.Error
	}
	return report.ID, nil
}

func (r *GormRepository) GetReport(id string) (*model.ModReport, error) {
	var report model.ModReport
	result := r.db.First(&report, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &report, nil
}

func (r *GormRepository) GetPendingReports() ([]model.ModReport, error) {
	var list []model.ModReport
	result := r.db.Where("status = ?", model.ReportStatusPending).Order("created_at ASC").Find(&list)
	if result.Error != nil {
		return nil, result.Error
	}
	return list, nil
}

func (r *GormRepository) GetReportsByModID(modID string) ([]model.ModReport, error) {
	var list []model.ModReport
	result := r.db.Where("mod_id = ?", modID).Order("created_at DESC").Find(&list)
	if result.Error != nil {
		return nil, result.Error
	}
	return list, nil
}

func (r *GormRepository) UpdateReport(report *model.ModReport) error {
	result := r.db.Save(report)
	return result.Error
}
