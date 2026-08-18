package musmgr

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ikafly144/au_mod_installer/server/model"
)

func TestNextVersionID(t *testing.T) {
	tests := []struct {
		existing []string
		expected string
	}{
		{existing: nil, expected: "1.0.0"},
		{existing: []string{}, expected: "1.0.0"},
		{existing: []string{"1.0.0"}, expected: "1.0.1"},
		{existing: []string{"v1.0.0", "v1.0.1", "v1.1.0"}, expected: "1.1.1"},
		{existing: []string{"0.9.5", "1.2.3-alpha"}, expected: "1.2.4"},
	}

	for _, tt := range tests {
		result := nextVersionID(tt.existing)
		assert.Equal(t, tt.expected, result)
	}
}

func TestParseFeatures(t *testing.T) {
	raw := []string{
		"direct_join=true",
		"legacy_mode=0",
		"custom_field=hello",
		"enabled=yes",
		"disabled=off",
	}

	features := parseFeatures(raw)
	assert.Equal(t, true, features["direct_join"])
	assert.Equal(t, false, features["legacy_mode"])
	assert.Equal(t, "hello", features["custom_field"])
	assert.Equal(t, true, features["enabled"])
	assert.Equal(t, false, features["disabled"])
}

func TestParseDependencies(t *testing.T) {
	raw := []string{
		"mod1:^1.0.0:required",
		"mod2:>=2.0.0:optional",
		"mod3:1.5.0",
	}

	deps := parseDependencies(raw)
	assert.Len(t, deps, 3)

	assert.Equal(t, "mod1", deps[0].ModID)
	assert.Equal(t, "^1.0.0", deps[0].VersionID)
	assert.Equal(t, model.DependencyTypeRequired, deps[0].DependencyType)

	assert.Equal(t, "mod2", deps[1].ModID)
	assert.Equal(t, ">=2.0.0", deps[1].VersionID)
	assert.Equal(t, model.DependencyTypeOptional, deps[1].DependencyType)

	assert.Equal(t, "mod3", deps[2].ModID)
	assert.Equal(t, "1.5.0", deps[2].VersionID)
	assert.Equal(t, model.DependencyTypeRequired, deps[2].DependencyType)
}

func TestParseFileFlag(t *testing.T) {
	t.Run("direct URL", func(t *testing.T) {
		pf := parseFileFlag("https://example.com/mod.zip")
		assert.Equal(t, []string{"https://example.com/mod.zip"}, pf.URLs)
		assert.Equal(t, string(model.FileTypeArchive), pf.Type)
		assert.Equal(t, string(model.TargetPlatformAny), pf.TargetPlatform)
	})

	t.Run("direct path", func(t *testing.T) {
		pf := parseFileFlag("C:/test/mod.dll")
		assert.Equal(t, "C:/test/mod.dll", pf.Path)
		assert.Equal(t, string(model.FileTypeArchive), pf.Type)
	})

	t.Run("structured key-value", func(t *testing.T) {
		pf := parseFileFlag("path=C:/test/mod.dll;type=plugin_dll;target_platform=x64;extract_path=plugins/mod.dll")
		assert.Equal(t, "C:/test/mod.dll", pf.Path)
		assert.Equal(t, "plugin_dll", pf.Type)
		assert.Equal(t, "x64", pf.TargetPlatform)
		assert.NotNil(t, pf.ExtractPath)
		assert.Equal(t, "plugins/mod.dll", *pf.ExtractPath)
	})
}
