//go:build windows

package aumgr

import (
	"fmt"
	"path/filepath"

	"github.com/ikafly144/au_mod_installer/pkg/assetstools"
)

func readVersionFile(globalGameManagersPath string) (string, error) {
	version, err := assetstools.ReadPlayerSettingsBundleVersion(globalGameManagersPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", globalGameManagersPath, err)
	}
	return version, nil
}

func GetVersion(gamePath string) (version string, err error) {
	versionFilePath := filepath.Join(gamePath, "Among Us_Data", "globalgamemanagers")
	return readVersionFile(versionFilePath)
}
