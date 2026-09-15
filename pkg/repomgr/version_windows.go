//go:build windows

package repomgr

import (
	"fmt"
	"path/filepath"

	"github.com/ikafly144/modrepo/pkg/assetstools"
)

func GetVersion(gamePath string) (string, error) {
	globalGameManagersPath := filepath.Join(gamePath, DataDirName, "globalgamemanagers")
	version, err := assetstools.ReadPlayerSettingsBundleVersion(globalGameManagersPath)
	if err != nil {
		return "", fmt.Errorf("failed to read game version from %s: %w", globalGameManagersPath, err)
	}
	return version, nil
}
