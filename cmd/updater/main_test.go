//go:build windows

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildLaunchArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty args",
			input:    []string{},
			expected: []string{"-initial"},
		},
		{
			name:     "with silent flag",
			input:    []string{"-silent"},
			expected: []string{"-initial", "-silent"},
		},
		{
			name:     "with existing initial flag",
			input:    []string{"-initial", "-silent"},
			expected: []string{"-initial", "-silent"},
		},
		{
			name:     "with uri and server flags",
			input:    []string{"-server", "http://example.com", "mod-of-us://join/123"},
			expected: []string{"-initial", "-server", "http://example.com", "mod-of-us://join/123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildLaunchArgs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

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
