package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorageService_GetPublicURL(t *testing.T) {
	// 1. MinIO / Local Path Style
	s1 := &s3StorageService{
		cfg: StorageConfig{
			Endpoint:       "http://localhost:9000",
			PublicEndpoint: "http://localhost:9000",
			Bucket:         "au-mods",
			UsePathStyle:   true,
		},
	}
	assert.Equal(t, "http://localhost:9000/au-mods/mods/au.coolmod/1.0.0/mod.zip", s1.GetPublicURL("mods/au.coolmod/1.0.0/mod.zip"))

	// 2. Custom CDN / Domain
	s2 := &s3StorageService{
		cfg: StorageConfig{
			Endpoint:       "https://account.r2.cloudflarestorage.com",
			PublicEndpoint: "https://cdn.modofus.com",
			Bucket:         "au-mods",
			UsePathStyle:   true,
		},
	}
	assert.Equal(t, "https://cdn.modofus.com/au-mods/plugins/mod.dll", s2.GetPublicURL("plugins/mod.dll"))

	// 3. AWS S3 Default fallback
	s3 := &s3StorageService{
		cfg: StorageConfig{
			Region: "ap-northeast-1",
			Bucket: "au-mods-production",
		},
	}
	assert.Equal(t, "https://au-mods-production.s3.ap-northeast-1.amazonaws.com/files/mod.zip", s3.GetPublicURL("files/mod.zip"))
}
