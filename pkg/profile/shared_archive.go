package profile

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	SharedArchiveProfilePath = "mod-of-us.profile.json"
	SharedArchiveIconPath    = "icon.png"

	maxSharedArchiveProfileSize = 1 << 20 // 1 MiB
	maxSharedArchiveIconSize    = 8 << 20 // 8 MiB
)

type sharedArchiveProfileDocument struct {
	SharedProfile json.RawMessage `json:"sharedprofile"`
}

func EncodeSharedArchive(shared SharedProfile, iconPNG []byte) ([]byte, error) {
	sharedJSON, err := json.Marshal(shared)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal shared profile: %w", err)
	}
	documentJSON, err := json.Marshal(sharedArchiveProfileDocument{
		SharedProfile: sharedJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal archive profile document: %w", err)
	}

	var buf bytes.Buffer
	archiveWriter := zip.NewWriter(&buf)

	profileWriter, err := archiveWriter.Create(SharedArchiveProfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s in archive: %w", SharedArchiveProfilePath, err)
	}
	if _, err := profileWriter.Write(documentJSON); err != nil {
		return nil, fmt.Errorf("failed to write %s in archive: %w", SharedArchiveProfilePath, err)
	}

	if len(iconPNG) > 0 {
		iconWriter, err := archiveWriter.Create(SharedArchiveIconPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create %s in archive: %w", SharedArchiveIconPath, err)
		}
		if _, err := iconWriter.Write(iconPNG); err != nil {
			return nil, fmt.Errorf("failed to write %s in archive: %w", SharedArchiveIconPath, err)
		}
	}

	if err := archiveWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize archive: %w", err)
	}

	return buf.Bytes(), nil
}

func DecodeSharedArchive(reader io.ReaderAt, size int64) (*SharedProfile, []byte, error) {
	if reader == nil {
		return nil, nil, fmt.Errorf("archive reader is nil")
	}

	archiveReader, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open profile archive: %w", err)
	}

	var profileJSON []byte
	var iconPNG []byte
	for _, f := range archiveReader.File {
		switch normalizeArchivePath(f.Name) {
		case SharedArchiveProfilePath:
			profileJSON, err = readZipEntryLimited(f, maxSharedArchiveProfileSize)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read %s: %w", SharedArchiveProfilePath, err)
			}
		case SharedArchiveIconPath:
			iconPNG, err = readZipEntryLimited(f, maxSharedArchiveIconSize)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read %s: %w", SharedArchiveIconPath, err)
			}
		}
	}

	if len(profileJSON) == 0 {
		return nil, nil, fmt.Errorf("%s is missing in archive", SharedArchiveProfilePath)
	}

	var document sharedArchiveProfileDocument
	if err := json.Unmarshal(profileJSON, &document); err != nil {
		return nil, nil, fmt.Errorf("failed to parse %s: %w", SharedArchiveProfilePath, err)
	}
	if len(document.SharedProfile) == 0 {
		return nil, nil, fmt.Errorf("sharedprofile is missing in %s", SharedArchiveProfilePath)
	}

	var shared SharedProfile
	if err := json.Unmarshal(document.SharedProfile, &shared); err != nil {
		return nil, nil, fmt.Errorf("failed to parse sharedprofile in %s: %w", SharedArchiveProfilePath, err)
	}

	return &shared, iconPNG, nil
}

func normalizeArchivePath(name string) string {
	path := strings.ReplaceAll(name, "\\", "/")
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "/")
	return path
}

func readZipEntryLimited(file *zip.File, maxBytes int64) ([]byte, error) {
	if file == nil {
		return nil, fmt.Errorf("zip entry is nil")
	}
	if file.UncompressedSize64 > uint64(maxBytes) {
		return nil, fmt.Errorf("entry too large")
	}

	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("entry too large")
	}
	return data, nil
}
