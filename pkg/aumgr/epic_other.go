//go:build !windows

package aumgr

import "errors"

func getEpicManifest() (Manifest, error) {
	return nil, errors.New("Epic Games manifest detection is only supported on Windows")
}
