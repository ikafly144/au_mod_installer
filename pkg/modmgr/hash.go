package modmgr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func hashModVersion(version ModVersion) (string, error) {
	hasher := sha256.New()
	err := json.NewEncoder(hasher).Encode(version)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
