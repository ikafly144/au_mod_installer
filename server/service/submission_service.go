package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/server/model"
	"github.com/ikafly144/au_mod_installer/server/repository"
)

var (
	ErrModAlreadyExists      = errors.New("a mod with this ID already exists")
	ErrSubmissionNotFound    = errors.New("submission not found")
	ErrUnauthorized          = errors.New("you do not have permission to manage this mod")
	ErrInvalidModID          = errors.New("invalid mod ID: only alphanumeric characters, dots, and hyphens are allowed")
	ErrInvalidVersionID      = errors.New("invalid version ID")
	ErrSuspiciousFilesFound  = errors.New("suspicious files detected in mod archive")
	validModIDRegex          = regexp.MustCompile(`^[a-zA-Z0-9_\.\-]+$`)
)

type SubmitModRequest struct {
	ModID        string
	Name         string
	Description  string
	AuthorName   string
	SubmitterID  string // Discord User ID
	ThumbnailURL string
}

type SubmitVersionRequest struct {
	ModID          string
	VersionID      string
	Changelog      string
	SubmitterID    string
	TargetPlatform model.TargetPlatform
	Filename       string
	FileData       []byte
	ExternalURL    string
	ExtractPath    *string
	Dependencies   model.DependencyArray
	Features       model.Features
}

type SubmissionService struct {
	subRepo repository.SubmissionRepository
	modRepo repository.ModRepository
	storage StorageService
	scanner ScanService
}

func NewSubmissionService(
	subRepo repository.SubmissionRepository,
	modRepo repository.ModRepository,
	storage StorageService,
	scanner ScanService,
) *SubmissionService {
	return &SubmissionService{
		subRepo: subRepo,
		modRepo: modRepo,
		storage: storage,
		scanner: scanner,
	}
}

func (s *SubmissionService) SubmitMod(ctx context.Context, req SubmitModRequest) (*model.ModSubmission, error) {
	req.ModID = strings.TrimSpace(req.ModID)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.AuthorName = strings.TrimSpace(req.AuthorName)

	if req.ModID == "" || !validModIDRegex.MatchString(req.ModID) {
		return nil, ErrInvalidModID
	}
	if req.Name == "" || req.AuthorName == "" {
		return nil, errors.New("mod name and author name are required")
	}

	// Check if already exists in production mod repository
	if existing, _ := s.modRepo.GetModDetails(req.ModID); existing != nil {
		return nil, ErrModAlreadyExists
	}

	submission := &model.ModSubmission{
		ID:           uuid.NewString(),
		ModID:        req.ModID,
		Name:         req.Name,
		Description:  req.Description,
		AuthorName:   req.AuthorName,
		SubmitterID:  req.SubmitterID,
		ThumbnailURL: req.ThumbnailURL,
		Status:       model.SubmissionStatusPendingReview,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err := s.subRepo.CreateModSubmission(submission)
	if err != nil {
		return nil, fmt.Errorf("failed to create mod submission: %w", err)
	}

	return submission, nil
}

func (s *SubmissionService) SubmitVersion(ctx context.Context, req SubmitVersionRequest) (*model.VersionSubmission, *InspectionResult, error) {
	req.ModID = strings.TrimSpace(req.ModID)
	req.VersionID = strings.TrimSpace(req.VersionID)
	if req.ModID == "" {
		return nil, nil, errors.New("mod_id is required")
	}
	if req.VersionID == "" {
		return nil, nil, ErrInvalidVersionID
	}

	// Check permissions on production mod if it exists
	mod, err := s.modRepo.GetModDetails(req.ModID)
	if err == nil && mod != nil {
		if !s.isAuthorized(mod, req.SubmitterID) {
			return nil, nil, ErrUnauthorized
		}
	}

	if len(req.FileData) == 0 && req.ExternalURL == "" {
		return nil, nil, errors.New("either file data or external download URL is required")
	}

	var inspection *InspectionResult
	hashes := make(model.StringMap)
	var downloadURLs model.StringArray
	fileSize := int64(len(req.FileData))
	contentType := model.FileTypeArchive

	if len(req.FileData) > 0 {
		var inspectErr error
		inspection, inspectErr = s.scanner.InspectFile(req.Filename, req.FileData)
		if inspectErr != nil {
			return nil, nil, fmt.Errorf("failed to inspect file: %w", inspectErr)
		}

		if len(inspection.SuspiciousFiles) > 0 {
			return nil, inspection, fmt.Errorf("%w: %s", ErrSuspiciousFilesFound, strings.Join(inspection.SuspiciousFiles, ", "))
		}

		hashes["sha256"] = inspection.SHA256
		hashes["md5"] = inspection.MD5

		ext := strings.ToLower(filepath.Ext(req.Filename))
		if ext == ".dll" {
			contentType = model.FileTypePluginDll
		}

		// Upload to S3 storage
		storageKey := fmt.Sprintf("mods/%s/%s/%s", req.ModID, req.VersionID, req.Filename)
		publicURL, uploadErr := s.storage.Upload(ctx, storageKey, req.FileData, "application/octet-stream")
		if uploadErr != nil {
			return nil, inspection, fmt.Errorf("failed to upload mod file to storage: %w", uploadErr)
		}
		downloadURLs = append(downloadURLs, publicURL)
	}

	if req.ExternalURL != "" {
		downloadURLs = append(downloadURLs, req.ExternalURL)
	}

	// VirusTotal lookup / scan
	vtStatus := "unscanned"
	vtScore := 0
	vtURL := ""
	if inspection != nil && inspection.SHA256 != "" {
		vtRes, vtErr := s.scanner.CheckVirusTotal(ctx, inspection.SHA256, req.FileData)
		if vtErr == nil && vtRes != nil {
			vtStatus = vtRes.Status
			vtScore = vtRes.Score
			vtURL = vtRes.Permalink
		}
	}

	if req.TargetPlatform == "" {
		req.TargetPlatform = model.TargetPlatformAny
	}

	verSub := &model.VersionSubmission{
		ID:               uuid.NewString(),
		ModID:            req.ModID,
		VersionID:        req.VersionID,
		Changelog:        req.Changelog,
		SubmitterID:      req.SubmitterID,
		Status:           model.SubmissionStatusPendingReview,
		TargetPlatform:   req.TargetPlatform,
		ContentType:      contentType,
		Filename:         req.Filename,
		FileSize:         fileSize,
		ExtractPath:      req.ExtractPath,
		Hashes:           hashes,
		Downloads:        downloadURLs,
		Dependencies:     req.Dependencies,
		Features:         req.Features,
		VirusTotalStatus: vtStatus,
		VirusTotalScore:  vtScore,
		VirusTotalURL:    vtURL,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	_, err = s.subRepo.CreateVersionSubmission(verSub)
	if err != nil {
		return nil, inspection, fmt.Errorf("failed to create version submission: %w", err)
	}

	return verSub, inspection, nil
}

func (s *SubmissionService) ApproveModSubmission(ctx context.Context, submissionID, reviewerID string) (*model.ModDetails, error) {
	sub, err := s.subRepo.GetModSubmission(submissionID)
	if err != nil {
		return nil, ErrSubmissionNotFound
	}
	if sub.Status == model.SubmissionStatusApproved {
		return nil, errors.New("submission is already approved")
	}

	now := time.Now()
	sub.Status = model.SubmissionStatusApproved
	sub.ReviewedBy = reviewerID
	sub.ReviewedAt = &now
	if err := s.subRepo.UpdateModSubmission(sub); err != nil {
		return nil, fmt.Errorf("failed to update submission status: %w", err)
	}

	var thumb *string
	if sub.ThumbnailURL != "" {
		thumb = &sub.ThumbnailURL
	}

	mod := &model.ModDetails{
		ID:                     sub.ModID,
		Name:                   sub.Name,
		Description:            sub.Description,
		Author:                 sub.AuthorName,
		OwnerDiscordID:         sub.SubmitterID,
		CollaboratorDiscordIDs: model.StringArray{},
		ThumbnailURI:           thumb,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	_, err = s.modRepo.CreateMod(mod)
	if err != nil {
		return nil, fmt.Errorf("failed to create production mod: %w", err)
	}

	return mod, nil
}

func (s *SubmissionService) RejectModSubmission(ctx context.Context, submissionID, reviewerID, reason string) (*model.ModSubmission, error) {
	sub, err := s.subRepo.GetModSubmission(submissionID)
	if err != nil {
		return nil, ErrSubmissionNotFound
	}

	now := time.Now()
	sub.Status = model.SubmissionStatusRejected
	sub.ReviewedBy = reviewerID
	sub.ReviewedAt = &now
	sub.RejectionReason = reason

	if err := s.subRepo.UpdateModSubmission(sub); err != nil {
		return nil, fmt.Errorf("failed to update submission status: %w", err)
	}

	return sub, nil
}

func (s *SubmissionService) ApproveVersionSubmission(ctx context.Context, verSubmissionID, reviewerID string) (*model.ModVersionDetails, error) {
	verSub, err := s.subRepo.GetVersionSubmission(verSubmissionID)
	if err != nil {
		return nil, ErrSubmissionNotFound
	}
	if verSub.Status == model.SubmissionStatusApproved {
		return nil, errors.New("version submission is already approved")
	}

	now := time.Now()
	verSub.Status = model.SubmissionStatusApproved
	verSub.ReviewedBy = reviewerID
	verSub.ReviewedAt = &now
	if err := s.subRepo.UpdateVersionSubmission(verSub); err != nil {
		return nil, fmt.Errorf("failed to update version submission status: %w", err)
	}

	versionInternalID := uuid.NewString()
	fileID := uuid.NewString()

	verFile := model.ModVersionFile{
		ID:             fileID,
		ModID:          &verSub.ModID,
		VersionID:      &versionInternalID,
		Filename:       verSub.Filename,
		ContentType:    verSub.ContentType,
		Size:           verSub.FileSize,
		ExtractPath:    verSub.ExtractPath,
		TargetPlatform: verSub.TargetPlatform,
		Hashes:         verSub.Hashes,
		Downloads:      verSub.Downloads,
		CreatedAt:      now,
	}

	versionDetails := &model.ModVersionDetails{
		ID:           versionInternalID,
		VersionID:    verSub.VersionID,
		ModID:        verSub.ModID,
		Files:        []model.ModVersionFile{verFile},
		Dependencies: verSub.Dependencies,
		Features:     verSub.Features,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err = s.modRepo.CreateModVersion(verSub.ModID, versionDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to create production mod version: %w", err)
	}

	// Update latest version ID on ModDetails
	if err := s.modRepo.UpdateMod(verSub.ModID, &model.ModDetails{
		LatestVersionID: &versionInternalID,
	}); err != nil {
		// Log or handle non-fatal update
	}

	return versionDetails, nil
}

func (s *SubmissionService) RejectVersionSubmission(ctx context.Context, verSubmissionID, reviewerID, reason string) (*model.VersionSubmission, error) {
	verSub, err := s.subRepo.GetVersionSubmission(verSubmissionID)
	if err != nil {
		return nil, ErrSubmissionNotFound
	}

	now := time.Now()
	verSub.Status = model.SubmissionStatusRejected
	verSub.ReviewedBy = reviewerID
	verSub.ReviewedAt = &now
	verSub.RejectionReason = reason

	if err := s.subRepo.UpdateVersionSubmission(verSub); err != nil {
		return nil, fmt.Errorf("failed to update version submission status: %w", err)
	}

	return verSub, nil
}

func (s *SubmissionService) GetModSubmission(id string) (*model.ModSubmission, error) {
	return s.subRepo.GetModSubmission(id)
}

func (s *SubmissionService) GetVersionSubmission(id string) (*model.VersionSubmission, error) {
	return s.subRepo.GetVersionSubmission(id)
}

func (s *SubmissionService) AddCollaborator(ctx context.Context, modID, requesterID, collaboratorID string) error {
	mod, err := s.modRepo.GetModDetails(modID)
	if err != nil {
		return err
	}
	if mod.OwnerDiscordID != requesterID {
		return ErrUnauthorized
	}

	if !slices.Contains(mod.CollaboratorDiscordIDs, collaboratorID) {
		mod.CollaboratorDiscordIDs = append(mod.CollaboratorDiscordIDs, collaboratorID)
		return s.modRepo.UpdateMod(modID, &model.ModDetails{
			CollaboratorDiscordIDs: mod.CollaboratorDiscordIDs,
		})
	}
	return nil
}

func (s *SubmissionService) RemoveCollaborator(ctx context.Context, modID, requesterID, collaboratorID string) error {
	mod, err := s.modRepo.GetModDetails(modID)
	if err != nil {
		return err
	}
	if mod.OwnerDiscordID != requesterID {
		return ErrUnauthorized
	}

	var newCollabs model.StringArray
	for _, id := range mod.CollaboratorDiscordIDs {
		if id != collaboratorID {
			newCollabs = append(newCollabs, id)
		}
	}
	mod.CollaboratorDiscordIDs = newCollabs
	return s.modRepo.UpdateMod(modID, &model.ModDetails{
		CollaboratorDiscordIDs: mod.CollaboratorDiscordIDs,
	})
}

func (s *SubmissionService) TransferOwnership(ctx context.Context, modID, currentOwnerID, newOwnerID string) error {
	mod, err := s.modRepo.GetModDetails(modID)
	if err != nil {
		return err
	}
	if mod.OwnerDiscordID != currentOwnerID {
		return ErrUnauthorized
	}

	mod.OwnerDiscordID = newOwnerID
	return s.modRepo.UpdateMod(modID, &model.ModDetails{
		OwnerDiscordID: newOwnerID,
	})
}

func (s *SubmissionService) SetDiscordThreadID(ctx context.Context, modID, threadID string) error {
	return s.modRepo.UpdateMod(modID, &model.ModDetails{
		DiscordThreadID: threadID,
	})
}

func (s *SubmissionService) isAuthorized(mod *model.ModDetails, discordUserID string) bool {
	if mod.OwnerDiscordID == "" || mod.OwnerDiscordID == discordUserID {
		return true
	}
	return slices.Contains(mod.CollaboratorDiscordIDs, discordUserID)
}
