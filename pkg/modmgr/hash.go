package modmgr

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha3"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"maps"

	"github.com/ikafly144/au_mod_installer/common/rest/model"
)

func hashModVersion(version ModVersion) (string, error) {
	hasher := sha256.New()
	err := json.NewEncoder(hasher).Encode(version)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func checkDownloadedFileHash(download *model.ModVersionFile) HashCheckingWriter {
	multi := make(map[string]HashCheckingWriter)
	for hashType, hashStr := range download.Hashes {
		var hasher hash.Hash
		switch hashType {
		case "md5":
			hasher = md5.New()
		case "sha1":
			hasher = sha1.New()
		case "sha256":
			hasher = sha256.New()
		case "sha512":
			hasher = sha512.New()
		case "sha384":
			hasher = sha512.New384()
		case "sha3-224":
			hasher = sha3.New224()
		case "sha3-256":
			hasher = sha3.New256()
		case "sha3-512":
			hasher = sha3.New512()
		case "sha3-384":
			hasher = sha3.New384()
		default:
			continue
		}
		if len(download.Hashes) == 1 {
			return &hashCheckingWriter{
				hasher:   hasher,
				hashType: hashType,
				hashStr:  hashStr,
			}
		}
		multi[hashType] = &hashCheckingWriter{
			hasher:   hasher,
			hashType: hashType,
			hashStr:  hashStr,
		}
	}
	return &multipleHashCheckingWriter{checkers: multi}
}

type HashCheckingWriter interface {
	io.Writer
	Sum() (map[string]string, error)
}

type multipleHashCheckingWriter struct {
	checkers map[string]HashCheckingWriter
}

func (w *multipleHashCheckingWriter) Write(p []byte) (n int, err error) {
	for _, checker := range w.checkers {
		if _, err := checker.Write(p); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (w *multipleHashCheckingWriter) Sum() (map[string]string, error) {
	if len(w.checkers) == 0 {
		return nil, fmt.Errorf("no valid hash checkers available")
	}
	hashes := make(map[string]string)
	for _, checker := range w.checkers {
		hash, err := checker.Sum()
		if err != nil {
			return nil, err
		}
		maps.Copy(hashes, hash)
	}
	return hashes, nil
}

type hashCheckingWriter struct {
	hasher   hash.Hash
	hashType string
	hashStr  string
}

func (w *hashCheckingWriter) Write(p []byte) (n int, err error) {
	n, err = w.hasher.Write(p)
	if err != nil {
		return n, err
	}
	return n, nil
}

func (w *hashCheckingWriter) Sum() (map[string]string, error) {
	calculatedHash := hex.EncodeToString(w.hasher.Sum(nil))
	if calculatedHash != w.hashStr {
		return nil, fmt.Errorf("hash mismatch: expected %s, got %s", w.hashStr, calculatedHash)
	}
	return map[string]string{w.hashType: calculatedHash}, nil
}
