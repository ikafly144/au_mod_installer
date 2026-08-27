package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type InspectionResult struct {
	SHA256          string
	MD5             string
	FileSize        int64
	IsArchive       bool
	ContainsDLL     bool
	DLLFiles        []string
	AllFiles        []string
	SuspiciousFiles []string
	Warnings        []string
}

type VirusTotalResult struct {
	Status    string // "clean", "suspicious", "malicious", "unscanned", "error"
	Score     int    // number of positive/malicious detections
	Total     int    // total number of scanning engines
	Permalink string
}

type ScanService interface {
	InspectFile(filename string, data []byte) (*InspectionResult, error)
	CheckVirusTotal(ctx context.Context, sha256Hash string, fileData []byte) (*VirusTotalResult, error)
}

type scanService struct {
	vtAPIKey   string
	httpClient *http.Client
}

func NewScanService(vtAPIKey string) ScanService {
	return &scanService{
		vtAPIKey: vtAPIKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

var bannedExtensions = map[string]bool{
	".exe": true,
	".bat": true,
	".cmd": true,
	".ps1": true,
	".vbs": true,
	".scr": true,
	".com": true,
	".pif": true,
	".sh":  true,
}

func (s *scanService) InspectFile(filename string, data []byte) (*InspectionResult, error) {
	if len(data) == 0 {
		return nil, errors.New("file data is empty")
	}

	// Calculate hashes
	sha256Hash := sha256.Sum256(data)
	md5Hash := md5.Sum(data)

	res := &InspectionResult{
		SHA256:   hex.EncodeToString(sha256Hash[:]),
		MD5:      hex.EncodeToString(md5Hash[:]),
		FileSize: int64(len(data)),
		DLLFiles: make([]string, 0),
		AllFiles: make([]string, 0),
		Warnings: make([]string, 0),
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".zip" || ext == ".aupack" || isZipHeader(data) {
		res.IsArchive = true
		reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, fmt.Errorf("failed to read zip archive: %w", err)
		}

		var totalUncompressedSize int64
		for _, f := range reader.File {
			// Check zip slip
			cleanName := filepath.Clean(f.Name)
			if strings.HasPrefix(cleanName, "..") || strings.HasPrefix(cleanName, "/") || strings.HasPrefix(cleanName, "\\") {
				res.SuspiciousFiles = append(res.SuspiciousFiles, f.Name)
				res.Warnings = append(res.Warnings, fmt.Sprintf("Zip slip detected in entry: %s", f.Name))
				continue
			}

			res.AllFiles = append(res.AllFiles, f.Name)
			totalUncompressedSize += int64(f.UncompressedSize64)

			entryExt := strings.ToLower(filepath.Ext(f.Name))
			if entryExt == ".dll" {
				res.ContainsDLL = true
				res.DLLFiles = append(res.DLLFiles, f.Name)
			}
			if bannedExtensions[entryExt] {
				res.SuspiciousFiles = append(res.SuspiciousFiles, f.Name)
				res.Warnings = append(res.Warnings, fmt.Sprintf("Suspicious executable entry: %s", f.Name))
			}
		}

		// Zip bomb protection check: e.g. uncompressed > 500MB
		if totalUncompressedSize > 500*1024*1024 {
			res.Warnings = append(res.Warnings, fmt.Sprintf("Total uncompressed size too large: %d MB", totalUncompressedSize/(1024*1024)))
		}
	} else if ext == ".dll" {
		res.ContainsDLL = true
		res.DLLFiles = append(res.DLLFiles, filename)
		res.AllFiles = append(res.AllFiles, filename)
	} else {
		res.AllFiles = append(res.AllFiles, filename)
	}

	return res, nil
}

func isZipHeader(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	return data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04
}

type vtFileResponse struct {
	Data struct {
		Attributes struct {
			LastAnalysisStats struct {
				Harmless   int `json:"harmless"`
				TypeUnsup  int `json:"type-unsupported"`
				Suspicious int `json:"suspicious"`
				Confirmed  int `json:"confirmed-timeout"`
				Timeout    int `json:"timeout"`
				Failure    int `json:"failure"`
				Malicious  int `json:"malicious"`
				Undetected int `json:"undetected"`
			} `json:"last_analysis_stats"`
		} `json:"attributes"`
	} `json:"data"`
}

func (s *scanService) CheckVirusTotal(ctx context.Context, sha256Hash string, fileData []byte) (*VirusTotalResult, error) {
	if s.vtAPIKey == "" {
		return &VirusTotalResult{
			Status:    "unscanned",
			Score:     0,
			Total:     0,
			Permalink: "",
		}, nil
	}

	url := fmt.Sprintf("https://www.virustotal.com/api/v3/files/%s", sha256Hash)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apikey", s.vtAPIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "Failed to query VirusTotal", "error", err, "hash", sha256Hash)
		return &VirusTotalResult{Status: "error", Permalink: fmt.Sprintf("https://www.virustotal.com/gui/file/%s", sha256Hash)}, nil
	}
	defer resp.Body.Close()

	permalink := fmt.Sprintf("https://www.virustotal.com/gui/file/%s", sha256Hash)

	if resp.StatusCode == http.StatusNotFound {
		// File is not yet scanned on VT
		return &VirusTotalResult{
			Status:    "unscanned",
			Score:     0,
			Total:     0,
			Permalink: permalink,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "VirusTotal returned non-200 status", "status", resp.StatusCode, "hash", sha256Hash)
		return &VirusTotalResult{Status: "error", Permalink: permalink}, nil
	}

	var vtResp vtFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&vtResp); err != nil {
		return nil, fmt.Errorf("failed to decode VirusTotal response: %w", err)
	}

	stats := vtResp.Data.Attributes.LastAnalysisStats
	maliciousCount := stats.Malicious
	suspiciousCount := stats.Suspicious
	totalEngines := stats.Harmless + stats.Undetected + stats.Malicious + stats.Suspicious

	status := "clean"
	if maliciousCount > 0 {
		status = "malicious"
	} else if suspiciousCount > 0 {
		status = "suspicious"
	}

	return &VirusTotalResult{
		Status:    status,
		Score:     maliciousCount + suspiciousCount,
		Total:     totalEngines,
		Permalink: permalink,
	}, nil
}
