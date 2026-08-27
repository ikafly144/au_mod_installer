package service

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ikafly144/au_mod_installer/server/model"
)

// Mock Storage
type mockStorage struct {
	uploaded map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{uploaded: make(map[string][]byte)}
}

func (m *mockStorage) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	m.uploaded[key] = data
	return "http://storage.local/" + key, nil
}

func (m *mockStorage) UploadReader(ctx context.Context, key string, body io.Reader, size int64, contentType string) (string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	m.uploaded[key] = data
	return "http://storage.local/" + key, nil
}

func (m *mockStorage) Download(ctx context.Context, key string) ([]byte, error) {
	if data, ok := m.uploaded[key]; ok {
		return data, nil
	}
	return nil, errors.New("not found")
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	delete(m.uploaded, key)
	return nil
}

func (m *mockStorage) GetPublicURL(key string) string {
	return "http://storage.local/" + key
}

func (m *mockStorage) EnsureBucket(ctx context.Context) error {
	return nil
}

// Mock ModRepository
type mockModRepo struct {
	mods     map[string]*model.ModDetails
	versions map[string]*model.ModVersionDetails
}

func newMockModRepo() *mockModRepo {
	return &mockModRepo{
		mods:     make(map[string]*model.ModDetails),
		versions: make(map[string]*model.ModVersionDetails),
	}
}

func (m *mockModRepo) CreateMod(details *model.ModDetails) (string, error) {
	m.mods[details.ID] = details
	return details.ID, nil
}

func (m *mockModRepo) CreateModVersion(modID string, details *model.ModVersionDetails) (string, error) {
	key := modID + "/" + details.VersionID
	m.versions[key] = details
	return details.VersionID, nil
}

func (m *mockModRepo) GetModIds(next string, limit int) (ids []string, nextID string, err error) {
	for k, mod := range m.mods {
		if mod.Status == model.ModStatusApproved || mod.Status == "" {
			ids = append(ids, k)
		}
	}
	return ids, "", nil
}

func (m *mockModRepo) GetModDetails(modID string) (*model.ModDetails, error) {
	if mod, ok := m.mods[modID]; ok {
		return mod, nil
	}
	return nil, errors.New("not found")
}

func (m *mockModRepo) GetModVersionIds(modID string) ([]string, error) {
	var ids []string
	for k := range m.versions {
		ids = append(ids, k)
	}
	return ids, nil
}

func (m *mockModRepo) GetModVersionDetails(modID, versionID string) (*model.ModVersionDetails, error) {
	key := modID + "/" + versionID
	if v, ok := m.versions[key]; ok {
		return v, nil
	}
	return nil, errors.New("not found")
}

func (m *mockModRepo) UpdateMod(modID string, details *model.ModDetails) error {
	if mod, ok := m.mods[modID]; ok {
		if details.Name != "" {
			mod.Name = details.Name
		}
		if details.Description != "" {
			mod.Description = details.Description
		}
		if details.Status != "" {
			mod.Status = details.Status
		}
		if details.OwnerDiscordID != "" {
			mod.OwnerDiscordID = details.OwnerDiscordID
		}
		if details.CollaboratorDiscordIDs != nil {
			mod.CollaboratorDiscordIDs = details.CollaboratorDiscordIDs
		}
		if details.DiscordThreadID != "" {
			mod.DiscordThreadID = details.DiscordThreadID
		}
		if details.LatestVersionID != nil {
			mod.LatestVersionID = details.LatestVersionID
		}
		return nil
	}
	return errors.New("not found")
}

func (m *mockModRepo) UpdateModVersion(modID, versionID string, details *model.ModVersionDetails) error {
	return nil
}

func (m *mockModRepo) DeleteMod(modID string) error {
	delete(m.mods, modID)
	return nil
}

func (m *mockModRepo) DeleteModVersion(modID, versionID string) error {
	delete(m.versions, modID+"/"+versionID)
	return nil
}

// Mock SubmissionRepository
type mockSubRepo struct {
	modSubs map[string]*model.ModSubmission
	verSubs map[string]*model.VersionSubmission
}

func newMockSubRepo() *mockSubRepo {
	return &mockSubRepo{
		modSubs: make(map[string]*model.ModSubmission),
		verSubs: make(map[string]*model.VersionSubmission),
	}
}

func (m *mockSubRepo) CreateModSubmission(sub *model.ModSubmission) (string, error) {
	m.modSubs[sub.ID] = sub
	return sub.ID, nil
}

func (m *mockSubRepo) GetModSubmission(id string) (*model.ModSubmission, error) {
	if s, ok := m.modSubs[id]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func (m *mockSubRepo) GetModSubmissionsByModID(modID string) ([]model.ModSubmission, error) {
	var res []model.ModSubmission
	for _, s := range m.modSubs {
		if s.ModID == modID {
			res = append(res, *s)
		}
	}
	return res, nil
}

func (m *mockSubRepo) GetPendingModSubmissions() ([]model.ModSubmission, error) {
	var res []model.ModSubmission
	for _, s := range m.modSubs {
		if s.Status == model.SubmissionStatusPendingReview {
			res = append(res, *s)
		}
	}
	return res, nil
}

func (m *mockSubRepo) GetUserModSubmissions(submitterID string) ([]model.ModSubmission, error) {
	var res []model.ModSubmission
	for _, s := range m.modSubs {
		if s.SubmitterID == submitterID {
			res = append(res, *s)
		}
	}
	return res, nil
}

func (m *mockSubRepo) UpdateModSubmission(sub *model.ModSubmission) error {
	m.modSubs[sub.ID] = sub
	return nil
}

func (m *mockSubRepo) DeleteModSubmission(id string) error {
	delete(m.modSubs, id)
	return nil
}

func (m *mockSubRepo) CreateVersionSubmission(sub *model.VersionSubmission) (string, error) {
	m.verSubs[sub.ID] = sub
	return sub.ID, nil
}

func (m *mockSubRepo) GetVersionSubmission(id string) (*model.VersionSubmission, error) {
	if s, ok := m.verSubs[id]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func (m *mockSubRepo) GetVersionSubmissionsByModID(modID string) ([]model.VersionSubmission, error) {
	var res []model.VersionSubmission
	for _, s := range m.verSubs {
		if s.ModID == modID {
			res = append(res, *s)
		}
	}
	return res, nil
}

func (m *mockSubRepo) GetPendingVersionSubmissions(modID string) ([]model.VersionSubmission, error) {
	var res []model.VersionSubmission
	for _, s := range m.verSubs {
		if s.Status == model.SubmissionStatusPendingReview {
			if modID == "" || s.ModID == modID {
				res = append(res, *s)
			}
		}
	}
	return res, nil
}

func (m *mockSubRepo) GetUserVersionSubmissions(submitterID string) ([]model.VersionSubmission, error) {
	var res []model.VersionSubmission
	for _, s := range m.verSubs {
		if s.SubmitterID == submitterID {
			res = append(res, *s)
		}
	}
	return res, nil
}

func (m *mockSubRepo) UpdateVersionSubmission(sub *model.VersionSubmission) error {
	m.verSubs[sub.ID] = sub
	return nil
}

func (m *mockSubRepo) DeleteVersionSubmission(id string) error {
	delete(m.verSubs, id)
	return nil
}

// Mock ModerationRepository
type mockModerationRepo struct {
	auditLogs []model.ModAuditLog
	reports   map[string]*model.ModReport
}

func newMockModerationRepo() *mockModerationRepo {
	return &mockModerationRepo{
		reports: make(map[string]*model.ModReport),
	}
}

func (m *mockModerationRepo) CreateAuditLog(log *model.ModAuditLog) error {
	m.auditLogs = append(m.auditLogs, *log)
	return nil
}

func (m *mockModerationRepo) GetAuditLogs(modID string, limit int) ([]model.ModAuditLog, error) {
	var res []model.ModAuditLog
	for _, l := range m.auditLogs {
		if modID == "" || l.ModID == modID {
			res = append(res, l)
		}
	}
	return res, nil
}

func (m *mockModerationRepo) CreateReport(report *model.ModReport) (string, error) {
	m.reports[report.ID] = report
	return report.ID, nil
}

func (m *mockModerationRepo) GetReport(id string) (*model.ModReport, error) {
	if r, ok := m.reports[id]; ok {
		return r, nil
	}
	return nil, errors.New("not found")
}

func (m *mockModerationRepo) GetPendingReports() ([]model.ModReport, error) {
	var res []model.ModReport
	for _, r := range m.reports {
		if r.Status == model.ReportStatusPending {
			res = append(res, *r)
		}
	}
	return res, nil
}

func (m *mockModerationRepo) GetReportsByModID(modID string) ([]model.ModReport, error) {
	var res []model.ModReport
	for _, r := range m.reports {
		if r.ModID == modID {
			res = append(res, *r)
		}
	}
	return res, nil
}

func (m *mockModerationRepo) UpdateReport(report *model.ModReport) error {
	m.reports[report.ID] = report
	return nil
}

func TestSubmissionService_Lifecycle(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	modRepo := newMockModRepo()
	subRepo := newMockSubRepo()
	modrRepo := newMockModerationRepo()
	scanner := NewScanService("")

	svc := NewSubmissionService(subRepo, modRepo, modrRepo, storage, scanner)

	// 1. Submit Mod
	sub, err := svc.SubmitMod(ctx, SubmitModRequest{
		ModID:       "com.example.testmod",
		Name:        "Test Mod",
		Description: "A great test mod",
		AuthorName:  "TestAuthor",
		SubmitterID: "user_12345",
	})
	require.NoError(t, err)
	assert.Equal(t, model.SubmissionStatusPendingReview, sub.Status)

	// 2. Approve Mod
	mod, err := svc.ApproveModSubmission(ctx, sub.ID, "staff_999")
	require.NoError(t, err)
	assert.Equal(t, "com.example.testmod", mod.ID)
	assert.Equal(t, "user_12345", mod.OwnerDiscordID)
	assert.Equal(t, model.ModStatusApproved, mod.Status)

	// Verify mod exists in production repo
	prodMod, err := modRepo.GetModDetails("com.example.testmod")
	require.NoError(t, err)
	assert.Equal(t, "Test Mod", prodMod.Name)

	// 3. Submit Version
	dummyDLL := []byte("MZ\x90\x00dummy-mod-code")
	verSub, insp, err := svc.SubmitVersion(ctx, SubmitVersionRequest{
		ModID:          "com.example.testmod",
		VersionID:      "1.0.0",
		Changelog:      "Initial release",
		SubmitterID:    "user_12345",
		TargetPlatform: model.TargetPlatformX64,
		Filename:       "TestMod.dll",
		FileData:       dummyDLL,
	})
	require.NoError(t, err)
	require.NotNil(t, insp)
	assert.Equal(t, model.SubmissionStatusPendingReview, verSub.Status)
	assert.NotEmpty(t, verSub.Downloads)

	// 4. Approve Version
	verDetails, err := svc.ApproveVersionSubmission(ctx, verSub.ID, "staff_999")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", verDetails.VersionID)

	// Verify latest version updated on mod
	updatedMod, err := modRepo.GetModDetails("com.example.testmod")
	require.NoError(t, err)
	assert.NotNil(t, updatedMod.LatestVersionID)

	// 5. Collaborators
	err = svc.AddCollaborator(ctx, "com.example.testmod", "user_12345", "collab_678")
	require.NoError(t, err)
	assert.Contains(t, updatedMod.CollaboratorDiscordIDs, "collab_678")

	// Collaborator can submit new version
	verSub2, _, err := svc.SubmitVersion(ctx, SubmitVersionRequest{
		ModID:          "com.example.testmod",
		VersionID:      "1.1.0",
		SubmitterID:    "collab_678",
		TargetPlatform: model.TargetPlatformX64,
		Filename:       "TestMod.dll",
		FileData:       dummyDLL,
	})
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", verSub2.VersionID)

	// 6. Moderation: Ban Mod
	err = svc.BanMod(ctx, "com.example.testmod", "staff_999", "Malware detected")
	require.NoError(t, err)
	bannedMod, err := modRepo.GetModDetails("com.example.testmod")
	require.NoError(t, err)
	assert.Equal(t, model.ModStatusBanned, bannedMod.Status)

	// 7. Moderation: Unban Mod
	err = svc.UnbanMod(ctx, "com.example.testmod", "staff_999")
	require.NoError(t, err)
	unbannedMod, err := modRepo.GetModDetails("com.example.testmod")
	require.NoError(t, err)
	assert.Equal(t, model.ModStatusApproved, unbannedMod.Status)

	// 8. User Reporting
	report, err := svc.CreateReport(ctx, "com.example.testmod", "player_99", model.ReportCategoryCrash, "Crashes on startup")
	require.NoError(t, err)
	assert.Equal(t, model.ReportStatusPending, report.Status)

	// 9. Resolve Report
	resolvedReport, err := svc.ResolveReport(ctx, report.ID, "staff_999", "Fixed in v1.1.0")
	require.NoError(t, err)
	assert.Equal(t, model.ReportStatusResolved, resolvedReport.Status)

	// 10. Audit Logs
	logs, err := svc.GetAuditLogs(ctx, "com.example.testmod", 20)
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
}
