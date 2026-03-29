package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/google/go-github/v80/github"
)

type Dependency struct {
	ModID          string `json:"mod_id"`
	VersionID      string `json:"version_id"`
	DependencyType string `json:"dependency_type"`
}

type FileRule struct {
	From           string `json:"from"`
	Artifact       string `json:"artifact"`
	ContentType    string `json:"content_type"`
	ExtractPath    string `json:"extract_path"`
	TargetPlatform string `json:"target_platform"`
	// GitHub Actions specific fields
	Source       string `json:"source"`        // "release" or "actions" (default: "release")
	WorkflowID   string `json:"workflow_id"`   // Workflow name or ID (for actions)
	Branch       string `json:"branch"`        // Branch name (for actions, default: default branch)
	ArtifactName string `json:"artifact_name"` // Artifact name pattern (for actions)
	FilePattern  string `json:"file_pattern"`  // File pattern within artifact zip (for actions)
}

type Feature struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type Rule struct {
	ModID                 string       `json:"mod_id"`
	GithubRepo            string       `json:"github_repo"`
	ActionsVersionBuilder string       `json:"actions_version_builder,omitempty"`
	Dependencies          []Dependency `json:"dependencies"`
	Features              []Feature    `json:"features,omitempty"`
	Files                 []FileRule   `json:"files"`
}

// listWorkflowsFromRepo lists all workflows in a repository
func listWorkflowsFromRepo(ctx context.Context, client *github.Client, owner, repo string) error {
	workflows, _, err := client.Actions.ListWorkflows(ctx, owner, repo, nil)
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	type WorkflowInfo struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Path  string `json:"path"`
		State string `json:"state"`
	}

	var output []WorkflowInfo
	for _, wf := range workflows.Workflows {
		output = append(output, WorkflowInfo{
			ID:    wf.GetID(),
			Name:  wf.GetName(),
			Path:  wf.GetPath(),
			State: wf.GetState(),
		})
	}

	outJson, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(outJson))
	return nil
}

// listArtifactsFromWorkflow lists artifacts from workflow runs
func listArtifactsFromWorkflow(ctx context.Context, client *github.Client, owner, repo, workflowID, branch string) error {
	opts := &github.ListWorkflowRunsOptions{
		Branch: branch,
		ListOptions: github.ListOptions{
			PerPage: 10,
		},
	}

	var runs *github.WorkflowRuns
	var err error

	if workflowID != "" {
		runs, _, err = client.Actions.ListWorkflowRunsByFileName(ctx, owner, repo, workflowID, opts)
	} else {
		runs, _, err = client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, &github.ListWorkflowRunsOptions{
			Branch:      branch,
			ListOptions: github.ListOptions{PerPage: 10},
		})
	}

	if err != nil {
		return fmt.Errorf("failed to list workflow runs: %w", err)
	}

	type ArtifactInfo struct {
		RunID       int64  `json:"run_id"`
		RunName     string `json:"run_name"`
		ArtifactID  int64  `json:"artifact_id"`
		Name        string `json:"name"`
		SizeInBytes int64  `json:"size_in_bytes"`
		CreatedAt   string `json:"created_at"`
	}

	var output []ArtifactInfo
	for _, run := range runs.WorkflowRuns {
		artifacts, _, err := client.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, run.GetID(), nil)
		if err != nil {
			continue
		}
		for _, artifact := range artifacts.Artifacts {
			output = append(output, ArtifactInfo{
				RunID:       run.GetID(),
				RunName:     run.GetName(),
				ArtifactID:  artifact.GetID(),
				Name:        artifact.GetName(),
				SizeInBytes: artifact.GetSizeInBytes(),
				CreatedAt:   artifact.GetCreatedAt().String(),
			})
		}
	}

	outJson, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(outJson))
	return nil
}

// downloadAndExtractArtifact downloads a GitHub Actions artifact and extracts files matching the pattern
func downloadAndExtractArtifact(ctx context.Context, client *github.Client, owner, repo string, artifactID int64, filePattern string) ([]byte, string, error) {
	url, _, err := client.Actions.DownloadArtifact(ctx, owner, repo, artifactID, 5)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get artifact download URL: %w", err)
	}

	// Download the artifact zip
	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download artifact: %w", err)
	}
	defer resp.Body.Close()

	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read artifact data: %w", err)
	}

	// Extract files from zip
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read zip: %w", err)
	}

	var pattern *regexp.Regexp
	if filePattern != "" {
		pattern, err = regexp.Compile(filePattern)
		if err != nil {
			return nil, "", fmt.Errorf("invalid file pattern: %w", err)
		}
	}

	// Find matching file in zip
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		if pattern != nil && !pattern.MatchString(file.Name) {
			continue
		}

		// Found a matching file, extract it
		rc, err := file.Open()
		if err != nil {
			continue
		}

		data, err := io.ReadAll(rc)
		rc.Close()

		if err != nil {
			continue
		}

		return data, file.Name, nil
	}

	return nil, "", fmt.Errorf("no file matching pattern '%s' found in artifact", filePattern)
}

func findFirstActionsFileRule(rule Rule) *FileRule {
	for i := range rule.Files {
		source := rule.Files[i].Source
		if source == "" {
			source = "release"
		}
		if source == "actions" {
			return &rule.Files[i]
		}
	}
	return nil
}

func shortSHA(sha string) string {
	if len(sha) >= 7 {
		return sha[:7]
	}
	return sha
}

func sanitizeVersionID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}

	var b strings.Builder
	lastDash := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if isAlphaNum || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}

	out := strings.Trim(b.String(), "-._")
	if out == "" {
		return "unknown"
	}
	return out
}

func toInt(v any) (int64, error) {
	switch n := v.(type) {
	case int:
		return int64(n), nil
	case int64:
		return n, nil
	case float64:
		return int64(n), nil
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid integer: %s", n)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("unsupported numeric type: %T", v)
	}
}

func actionsTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b any) (int64, error) {
			ai, err := toInt(a)
			if err != nil {
				return 0, err
			}
			bi, err := toInt(b)
			if err != nil {
				return 0, err
			}
			return ai + bi, nil
		},
		"sub": func(a, b any) (int64, error) {
			ai, err := toInt(a)
			if err != nil {
				return 0, err
			}
			bi, err := toInt(b)
			if err != nil {
				return 0, err
			}
			return ai - bi, nil
		},
		"mul": func(a, b any) (int64, error) {
			ai, err := toInt(a)
			if err != nil {
				return 0, err
			}
			bi, err := toInt(b)
			if err != nil {
				return 0, err
			}
			return ai * bi, nil
		},
		"div": func(a, b any) (int64, error) {
			ai, err := toInt(a)
			if err != nil {
				return 0, err
			}
			bi, err := toInt(b)
			if err != nil {
				return 0, err
			}
			if bi == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return ai / bi, nil
		},
		"mod": func(a, b any) (int64, error) {
			ai, err := toInt(a)
			if err != nil {
				return 0, err
			}
			bi, err := toInt(b)
			if err != nil {
				return 0, err
			}
			if bi == 0 {
				return 0, fmt.Errorf("mod by zero")
			}
			return ai % bi, nil
		},
		"pad": func(v any, width int) (string, error) {
			vi, err := toInt(v)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%0*d", width, vi), nil
		},
	}
}

func convertLegacyActionsTemplate(tpl string) string {
	replacer := strings.NewReplacer(
		"{run_id}", "{{.RunID}}",
		"{run_number}", "{{.RunNumber}}",
		"{run_attempt}", "{{.RunAttempt}}",
		"{head_sha}", "{{.HeadSHA}}",
		"{short_sha}", "{{.ShortSHA}}",
		"{branch}", "{{.Branch}}",
		"{workflow}", "{{.Workflow}}",
		"{event}", "{{.Event}}",
		"{created_at}", "{{.CreatedAt}}",
	)
	return replacer.Replace(tpl)
}

func renderActionsVersionTemplate(versionTemplate string, data map[string]any) (string, error) {
	converted := convertLegacyActionsTemplate(versionTemplate)

	tpl, err := template.New("actions-version-builder").Option("missingkey=error").Funcs(actionsTemplateFuncMap()).Parse(converted)
	if err != nil {
		return "", fmt.Errorf("invalid actions version template: %w", err)
	}

	var b strings.Builder
	if err := tpl.Execute(&b, data); err != nil {
		return "", fmt.Errorf("failed to evaluate actions version template: %w", err)
	}

	return b.String(), nil
}

func buildActionsVersionID(run *github.WorkflowRun, explicitBranch, versionTemplate string) (string, error) {
	branch := strings.TrimSpace(explicitBranch)
	if branch == "" {
		branch = strings.TrimSpace(run.GetHeadBranch())
	}
	sha := strings.TrimSpace(run.GetHeadSHA())
	short := shortSHA(sha)
	if short == "" {
		short = "unknown"
	}

	versionTemplate = strings.TrimSpace(versionTemplate)
	if versionTemplate == "" {
		if branch != "" && branch != "main" && branch != "master" {
			return sanitizeVersionID(fmt.Sprintf("%s-%s", branch, short)), nil
		}
		return sanitizeVersionID(short), nil
	}

	createdAt := ""
	if t := run.GetCreatedAt(); !t.IsZero() {
		createdAt = t.Time.UTC().Format("20060102")
	}

	data := map[string]any{
		"RunID":      run.GetID(),
		"RunNumber":  run.GetRunNumber(),
		"RunAttempt": run.GetRunAttempt(),
		"HeadSHA":    sha,
		"ShortSHA":   short,
		"Branch":     branch,
		"Workflow":   run.GetName(),
		"Event":      run.GetEvent(),
		"CreatedAt":  createdAt,
	}
	versionID, err := renderActionsVersionTemplate(versionTemplate, data)
	if err != nil {
		return "", err
	}

	return sanitizeVersionID(versionID), nil
}

// fetchFromActions fetches version information from GitHub Actions artifacts
func fetchFromActions(ctx context.Context, client *github.Client, owner, repo string, rule Rule, workflowID, branch, versionTemplate string) (*Output, error) {
	firstActionsRule := findFirstActionsFileRule(rule)
	if firstActionsRule != nil {
		if workflowID == "" {
			workflowID = firstActionsRule.WorkflowID
		}
		if branch == "" {
			branch = firstActionsRule.Branch
		}
	}

	// Get latest successful workflow run
	opts := &github.ListWorkflowRunsOptions{
		Branch: branch,
		Status: "success",
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	}

	var runs *github.WorkflowRuns
	var err error

	if workflowID != "" {
		runs, _, err = client.Actions.ListWorkflowRunsByFileName(ctx, owner, repo, workflowID, opts)
	} else {
		runs, _, err = client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, &github.ListWorkflowRunsOptions{
			Branch:      branch,
			Status:      "success",
			ListOptions: github.ListOptions{PerPage: 1},
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list workflow runs: %w", err)
	}

	if len(runs.WorkflowRuns) == 0 {
		return nil, fmt.Errorf("no successful workflow runs found")
	}

	run := runs.WorkflowRuns[0]

	if versionTemplate == "" {
		versionTemplate = rule.ActionsVersionBuilder
	}
	versionID, err := buildActionsVersionID(run, branch, versionTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to build version id: %w", err)
	}

	out := &Output{
		ModID:     rule.ModID,
		VersionID: versionID,
	}

	// Get artifacts from the run
	artifacts, _, err := client.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, run.GetID(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}

	// Process file rules
	for _, fRule := range rule.Files {
		source := fRule.Source
		if source == "" {
			source = "release"
		}

		if source != "actions" {
			continue
		}

		fileStr := ""

		// Match artifact by name
		artifactPattern := fRule.ArtifactName
		if artifactPattern == "" {
			artifactPattern = fRule.Artifact // Fallback to artifact field
		}

		if artifactPattern == "" {
			continue
		}

		re, err := regexp.Compile(artifactPattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid artifact pattern '%s': %v\n", artifactPattern, err)
			continue
		}

		var matchedArtifact *github.Artifact
		for _, artifact := range artifacts.Artifacts {
			if re.MatchString(artifact.GetName()) {
				matchedArtifact = artifact
				break
			}
		}

		if matchedArtifact == nil {
			fmt.Fprintf(os.Stderr, "No artifact found matching pattern '%s'\n", artifactPattern)
			continue
		}

		artifactURL := &url.URL{
			Scheme: "gha",
			Host:   owner,
			Path:   fmt.Sprintf("/%s/artifact/%d", repo, matchedArtifact.GetID()),
		}
		if fRule.FilePattern != "" {
			q := artifactURL.Query()
			q.Set("file_pattern", fRule.FilePattern)
			artifactURL.RawQuery = q.Encode()
		}
		fileStr = fmt.Sprintf("url=%s", artifactURL.String())

		if fRule.ContentType != "" {
			fileStr += fmt.Sprintf(";type=%s", fRule.ContentType)
		}
		if fRule.ExtractPath != "" {
			fileStr += fmt.Sprintf(";extract_path=%s", fRule.ExtractPath)
		}
		if fRule.TargetPlatform != "" {
			fileStr += fmt.Sprintf(";target_platform=%s", fRule.TargetPlatform)
		}

		out.Files = append(out.Files, fileStr)
	}

	// Process dependencies
	for _, dep := range rule.Dependencies {
		dtype := dep.DependencyType
		if dtype == "" {
			return nil, fmt.Errorf("dependency %s does not specify dependency_type", dep.ModID)
		}
		out.Dependencies = append(out.Dependencies, fmt.Sprintf("%s:%s:%s", dep.ModID, dep.VersionID, dtype))
	}

	// Process features
	for _, feature := range rule.Features {
		name := strings.TrimSpace(feature.Name)
		if name == "" {
			return nil, fmt.Errorf("feature name cannot be empty")
		}
		value := feature.Value
		if value == nil {
			value = true
		}
		out.Features = append(out.Features, fmt.Sprintf("%s=%v", name, value))
	}

	return out, nil
}

type Output struct {
	ModID        string   `json:"mod_id"`
	VersionID    string   `json:"version_id"`
	Files        []string `json:"files"`
	Dependencies []string `json:"dependencies"`
	Features     []string `json:"features,omitempty"`
}

func main() {
	ruleFile := flag.String("rule", "", "Path to the rule file")
	listReleases := flag.Bool("list", false, "List available releases")
	tag := flag.String("tag", "", "Specific release tag to fetch")
	source := flag.String("source", "release", "Source type: 'release' or 'actions'")
	workflow := flag.String("workflow", "", "Workflow ID or name (for actions source)")
	branch := flag.String("branch", "", "Branch name (for actions source, default: default branch)")
	listWorkflows := flag.Bool("list-workflows", false, "List available workflows (for actions source)")
	listArtifacts := flag.Bool("list-artifacts", false, "List available artifacts (for actions source)")
	versionTemplate := flag.String("version-template", "", "Actions version template override (Go text/template). Fields: .RunNumber,.RunAttempt,.RunID,.ShortSHA,.HeadSHA,.Branch,.Workflow,.Event,.CreatedAt. Funcs: add,sub,mul,div,mod,pad")
	flag.Parse()

	if *ruleFile == "" {
		fmt.Fprintln(os.Stderr, "Usage: fetch-gh-release -rule <path-to-rule.json> [-list] [-tag <version>]")
		os.Exit(1)
	}

	data, err := os.ReadFile(*ruleFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read rule file: %v\n", err)
		os.Exit(1)
	}

	var rule Rule
	if err := json.Unmarshal(data, &rule); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse rule file: %v\n", err)
		os.Exit(1)
	}

	if rule.GithubRepo == "" {
		// Nothing to fetch
		return
	}

	parts := strings.SplitN(rule.GithubRepo, "/", 2)
	if len(parts) != 2 {
		fmt.Fprintf(os.Stderr, "Invalid github_repo format. Expected owner/repo, got: %s\n", rule.GithubRepo)
		os.Exit(1)
	}
	owner, repo := parts[0], parts[1]

	ctx := context.Background()
	client := github.NewClient(nil)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client = client.WithAuthToken(token)
	}

	// Handle Actions-specific listing commands
	if *listWorkflows {
		if err := listWorkflowsFromRepo(ctx, client, owner, repo); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *listArtifacts {
		if err := listArtifactsFromWorkflow(ctx, client, owner, repo, *workflow, *branch); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Handle source routing
	if *source == "actions" {
		out, err := fetchFromActions(ctx, client, owner, repo, rule, *workflow, *branch, *versionTemplate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch from actions: %v\n", err)
			os.Exit(1)
		}
		outJson, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(outJson))
		return
	}

	// Default: fetch from releases
	if *listReleases {
		releases, _, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{PerPage: 10})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list releases from %s: %v\n", rule.GithubRepo, err)
			os.Exit(1)
		}

		type ReleaseInfo struct {
			TagName     string `json:"tag_name"`
			Name        string `json:"name"`
			PublishedAt string `json:"published_at"`
		}

		var output []ReleaseInfo
		for _, r := range releases {
			output = append(output, ReleaseInfo{
				TagName:     r.GetTagName(),
				Name:        r.GetName(),
				PublishedAt: r.GetPublishedAt().String(),
			})
		}

		outJson, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(outJson))
		return
	}

	var release *github.RepositoryRelease

	if *tag != "" {
		release, _, err = client.Repositories.GetReleaseByTag(ctx, owner, repo, *tag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch release %s from %s: %v\n", *tag, rule.GithubRepo, err)
			os.Exit(1)
		}
	} else {
		release, _, err = client.Repositories.GetLatestRelease(ctx, owner, repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch latest release from %s: %v\n", rule.GithubRepo, err)
			os.Exit(1)
		}
	}

	out := Output{
		ModID:     rule.ModID,
		VersionID: strings.TrimPrefix(release.GetTagName(), "v"),
	}

	if len(rule.Files) == 0 {
		for _, asset := range release.Assets {
			out.Files = append(out.Files, fmt.Sprintf("url=%s", asset.GetBrowserDownloadURL()))
		}
	} else {
		for _, fRule := range rule.Files {
			// Skip actions-source rules when fetching from releases
			source := fRule.Source
			if source == "" {
				source = "release"
			}
			if source == "actions" {
				continue
			}

			fileStr := ""

			if fRule.From != "" {
				fileStr = fmt.Sprintf("url=%s", fRule.From)
			} else if fRule.Artifact != "" {
				re, err := regexp.Compile(fRule.Artifact)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Invalid artifact regex '%s': %v\n", fRule.Artifact, err)
					continue
				}
				matchedUrl := ""
				for _, asset := range release.Assets {
					if re.MatchString(asset.GetName()) {
						matchedUrl = asset.GetBrowserDownloadURL()
						break
					}
				}
				if matchedUrl != "" {
					fileStr = fmt.Sprintf("url=%s", matchedUrl)
				} else {
					fmt.Fprintf(os.Stderr, "No asset found matching regex '%s'\n", fRule.Artifact)
					continue
				}
			} else {
				continue
			}

			if fRule.ContentType != "" {
				fileStr += fmt.Sprintf(";type=%s", fRule.ContentType)
			}
			if fRule.ExtractPath != "" {
				fileStr += fmt.Sprintf(";extract_path=%s", fRule.ExtractPath)
			}
			if fRule.TargetPlatform != "" {
				fileStr += fmt.Sprintf(";target_platform=%s", fRule.TargetPlatform)
			}

			out.Files = append(out.Files, fileStr)
		}
	}

	for _, dep := range rule.Dependencies {
		dtype := dep.DependencyType
		if dtype == "" {
			fmt.Fprintf(os.Stderr, "Error: dependency %s does not specify dependency_type.\n", dep.ModID)
			os.Exit(1)
		}
		out.Dependencies = append(out.Dependencies, fmt.Sprintf("%s:%s:%s", dep.ModID, dep.VersionID, dtype))
	}
	for _, feature := range rule.Features {
		name := strings.TrimSpace(feature.Name)
		if name == "" {
			fmt.Fprintln(os.Stderr, "Error: feature name cannot be empty.")
			os.Exit(1)
		}
		value := feature.Value
		if value == nil {
			value = true
		}
		out.Features = append(out.Features, fmt.Sprintf("%s=%v", name, value))
	}

	outJson, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(outJson))
}
