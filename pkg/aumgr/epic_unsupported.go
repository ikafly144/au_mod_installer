//go:build !windows

package aumgr

import "errors"

func getEpicManifest() (Manifest, error) {
	return nil, errors.New("epic Games manifest detection is unsupported on this platform")
}
