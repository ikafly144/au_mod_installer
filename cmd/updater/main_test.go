//go:build windows

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldPerformUpdate(t *testing.T) {
	assert.True(t, shouldPerformUpdate("v1.0.0", "v1.1.0"))
	assert.True(t, shouldPerformUpdate("unknown", "v1.0.0"))
	assert.True(t, shouldPerformUpdate("", "v1.0.0"))
	assert.False(t, shouldPerformUpdate("v1.1.0", "v1.0.0"))
	assert.False(t, shouldPerformUpdate("v1.0.0", "v1.0.0"))
	assert.False(t, shouldPerformUpdate("v1.0.0", ""))
}

func TestReadUpdateBranchPreference(t *testing.T) {
	branch := readUpdateBranchPreference()
	assert.NotEmpty(t, branch)
}

func TestBuildLaunchArgs(t *testing.T) {
	assert.Equal(t, []string{"-silent", "-initial"}, buildLaunchArgs([]string{"-silent"}))
	assert.Equal(t, []string{"-silent", "-initial"}, buildLaunchArgs([]string{"-target", "Mod of Us.exe", "-silent"}))
	assert.Equal(t, []string{"-silent", "-initial"}, buildLaunchArgs([]string{"-target=Mod of Us.exe", "-silent"}))
	assert.Equal(t, []string{"-silent", "-initial"}, buildLaunchArgs([]string{"--target", "C:\\Mod of Us.exe", "-silent"}))
	assert.Equal(t, []string{"-silent", "-initial"}, buildLaunchArgs([]string{"-silent", "-initial"}))
}

func TestResolveTargetPath(t *testing.T) {
	absPath := `C:\Program Files\Mod of Us\Mod of Us.exe`
	assert.Equal(t, absPath, resolveTargetPath(absPath))
	assert.NotEmpty(t, resolveTargetPath(""))
}
