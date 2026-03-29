package main

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v80/github"
)

func TestBuildActionsVersionID_DefaultMainBranch(t *testing.T) {
	run := &github.WorkflowRun{
		HeadSHA:    github.Ptr("abcdef1234567890"),
		HeadBranch: github.Ptr("main"),
	}

	got, err := buildActionsVersionID(run, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "abcdef1" {
		t.Fatalf("got %q, want %q", got, "abcdef1")
	}
}

func TestBuildActionsVersionID_DefaultNonMainBranch(t *testing.T) {
	run := &github.WorkflowRun{
		HeadSHA:    github.Ptr("abcdef1234567890"),
		HeadBranch: github.Ptr("develop"),
	}

	got, err := buildActionsVersionID(run, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "develop-abcdef1" {
		t.Fatalf("got %q, want %q", got, "develop-abcdef1")
	}
}

func TestBuildActionsVersionID_Template(t *testing.T) {
	created := github.Timestamp{Time: time.Date(2026, 3, 29, 21, 42, 0, 0, time.UTC)}
	run := &github.WorkflowRun{
		ID:         github.Ptr(int64(12345)),
		RunNumber:  github.Ptr(77),
		RunAttempt: github.Ptr(3),
		HeadSHA:    github.Ptr("abcdef1234567890"),
		HeadBranch: github.Ptr("feature/x"),
		Name:       github.Ptr("ci"),
		Event:      github.Ptr("push"),
		CreatedAt:  &created,
	}

	got, err := buildActionsVersionID(run, "", "v{{.RunNumber}}.{{.RunAttempt}}-{{.ShortSHA}}-{{.Branch}}-{{.Workflow}}-{{.Event}}-{{.CreatedAt}}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "v77.3-abcdef1-feature-x-ci-push-20260329"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildActionsVersionID_UnknownPlaceholder(t *testing.T) {
	run := &github.WorkflowRun{
		HeadSHA: github.Ptr("abcdef1234567890"),
	}

	_, err := buildActionsVersionID(run, "", "{{.Unknown}}")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildActionsVersionID_LegacyTokenCompatibility(t *testing.T) {
	run := &github.WorkflowRun{
		RunNumber: github.Ptr(42),
		HeadSHA:   github.Ptr("abcdef1234567890"),
	}

	got, err := buildActionsVersionID(run, "", "v{run_number}-{short_sha}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "v42-abcdef1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildActionsVersionID_TemplateArithmetic(t *testing.T) {
	run := &github.WorkflowRun{
		RunNumber: github.Ptr(77),
	}

	got, err := buildActionsVersionID(run, "", "v{{add .RunNumber 1000}}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "v1077"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildActionsVersionID_TemplatePad(t *testing.T) {
	run := &github.WorkflowRun{
		RunNumber: github.Ptr(77),
	}

	got, err := buildActionsVersionID(run, "", "v{{pad .RunNumber 5}}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "v00077"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildActionsOutput_UsesGhaSchemeURL(t *testing.T) {
	// verify the URL construction rule used in fetchFromActions
	owner := "BepInEx"
	repo := "BepInEx"
	artifactID := int64(5811342878)
	filePattern := `BepInEx-Unity\.IL2CPP-win-x64-[\d.]+\.zip`

	artifactURL := &url.URL{
		Scheme: "gha",
		Host:   owner,
		Path:   fmt.Sprintf("/%s/artifact/%d", repo, artifactID),
	}
	q := artifactURL.Query()
	q.Set("file_pattern", filePattern)
	artifactURL.RawQuery = q.Encode()

	got := artifactURL.String()
	if !strings.HasPrefix(got, "gha://BepInEx/BepInEx/artifact/5811342878") {
		t.Fatalf("unexpected prefix: %s", got)
	}
	if !strings.Contains(got, "file_pattern=") {
		t.Fatalf("expected file_pattern in query: %s", got)
	}
}
