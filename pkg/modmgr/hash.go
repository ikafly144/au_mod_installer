package modmgr

import (
	"crypto/sha1"
	"encoding/hex"
)

func hashId(id string) string {
	// Simple hash function for demonstration purposes
	sha1Hasher := sha1.New()
	sha1Hasher.Write([]byte(id))
	return hex.EncodeToString(sha1Hasher.Sum(make([]byte, 0, 32)))
}
