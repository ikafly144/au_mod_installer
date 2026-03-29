package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/google/go-github/v80/github"
	"github.com/urfave/cli/v3"

	"github.com/ikafly144/au_mod_installer/common/rest/model"
)

type Dependency struct {
	ModID          string `json:"mod_id"`
	VersionID      string `json:"version_id"`
	DependencyType string `json:"dependency_type"`
}

type FileRule struct {
	From           string `json:"from,omitempty"`
	Artifact       string `json:"artifact,omitempty"`
	ContentType    string `json:"content_type"`
	ExtractPath    string `json:"extract_path,omitempty"`
	TargetPlatform string `json:"target_platform"`
	// GitHub Actions specific fields
	Source       string `json:"source,omitempty"`        // "release" or "actions"
	WorkflowID   string `json:"workflow_id,omitempty"`   // Workflow name or ID (for actions)
	Branch       string `json:"branch,omitempty"`        // Branch name (for actions)
	ArtifactName string `json:"artifact_name,omitempty"` // Artifact name pattern (for actions)
	FilePattern  string `json:"file_pattern,omitempty"`  // File pattern within artifact zip (for actions)
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

func main() {
	cmd := &cli.Command{
		Name:  "rule-gen",
		Usage: "Interactive rule generator for au_mod_installer",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return run(ctx)
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func contentTypeMap(s string) string {
	switch s {
	case "Archive":
		return string(model.ContentTypeArchive)
	case "Binary":
		return string(model.ContentTypeBinary)
	case "PluginDll":
		return string(model.ContentTypePluginDll)
	default:
		return string(model.ContentTypeArchive)
	}
}

func run(ctx context.Context) error {
	var repoInput string
	prompt := &survey.Input{
		Message: "Enter GitHub Repository (owner/repo):",
	}
	if err := survey.AskOne(prompt, &repoInput); err != nil {
		return err
	}

	parts := strings.Split(repoInput, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format (expected owner/repo)")
	}
	owner, repo := parts[0], parts[1]

	client := github.NewClient(nil)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client = client.WithAuthToken(token)
	}

	// Ask for source type
	var source string
	promptSource := &survey.Select{
		Message: "Select source for artifacts:",
		Options: []string{"Release", "Actions"},
		Default: "Release",
	}
	if err := survey.AskOne(promptSource, &source); err != nil {
		return err
	}

	var modID string
	var fileRules []FileRule
	var selectedAssets []string
	var workflowID int64
	var branch string
	var actionsVersionBuilder string

	if source == "Release" {
		fmt.Printf("Fetching latest release for %s/%s...\n", owner, repo)
		release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
		if err != nil {
			return fmt.Errorf("failed to fetch release: %w", err)
		}

		fmt.Printf("Found release: %s\n", release.GetTagName())

		promptModID := &survey.Input{
			Message: "Enter Mod ID:",
			Default: repo,
		}
		if err := survey.AskOne(promptModID, &modID); err != nil {
			return err
		}

		var assetOptions []string
		assetMap := make(map[string]*github.ReleaseAsset)

		for _, asset := range release.Assets {
			name := asset.GetName()
			assetOptions = append(assetOptions, name)
			assetMap[name] = asset
		}

		promptAssets := &survey.MultiSelect{
			Message: "Select assets to include in the rule:",
			Options: assetOptions,
		}
		if err := survey.AskOne(promptAssets, &selectedAssets); err != nil {
			return err
		}

		// Process release assets
		for _, assetName := range selectedAssets {
			asset := assetMap[assetName]
			fmt.Printf("\nConfiguring asset: %s\n", asset.GetName())

			fRule := FileRule{
				TargetPlatform: "any",
				ContentType:    string(model.ContentTypeArchive),
				Source:         "release",
			}

			var useRegex string
			promptRegex := &survey.Select{
				Message: "Use Artifact Regex or Explicit URL?",
				Options: []string{"Artifact Regex", "Explicit URL"},
				Default: "Artifact Regex",
			}
			if err := survey.AskOne(promptRegex, &useRegex); err != nil {
				return err
			}

			if useRegex == "Artifact Regex" {
				var artifactRegex string
				promptArtifact := &survey.Input{
					Message: "Enter artifact regex pattern:",
					Default: asset.GetName(),
				}
				if err := survey.AskOne(promptArtifact, &artifactRegex); err != nil {
					return err
				}
				fRule.Artifact = artifactRegex
			} else {
				fRule.From = asset.GetBrowserDownloadURL()
			}

			var contentType string
			promptContentType := &survey.Select{
				Message: "Select content type:",
				Options: []string{"Archive", "Binary", "PluginDll"},
				Default: "Archive",
			}
			if err := survey.AskOne(promptContentType, &contentType); err != nil {
				return err
			}
			fRule.ContentType = contentTypeMap(contentType)

			var extractPath string
			promptExtract := &survey.Input{
				Message: "Extract path (optional):",
			}
			if err := survey.AskOne(promptExtract, &extractPath); err != nil {
				return err
			}
			fRule.ExtractPath = extractPath

			var targetPlatform string
			promptPlatform := &survey.Select{
				Message: "Target platform:",
				Options: []string{"any", "windows", "linux", "darwin"},
				Default: "any",
			}
			if err := survey.AskOne(promptPlatform, &targetPlatform); err != nil {
				return err
			}
			fRule.TargetPlatform = targetPlatform

			fileRules = append(fileRules, fRule)
		}
	} else {
		// Actions workflow
		fmt.Printf("Fetching workflows for %s/%s...\n", owner, repo)
		workflows, _, err := client.Actions.ListWorkflows(ctx, owner, repo, nil)
		if err != nil {
			return fmt.Errorf("failed to fetch workflows: %w", err)
		}

		var workflowOptions []string
		workflowMap := make(map[string]*github.Workflow)
		for _, wf := range workflows.Workflows {
			if wf.GetState() == "active" {
				name := wf.GetName()
				workflowOptions = append(workflowOptions, name)
				workflowMap[name] = wf
			}
		}

		if len(workflowOptions) == 0 {
			return fmt.Errorf("no active workflows found")
		}

		var selectedWorkflow string
		promptWorkflow := &survey.Select{
			Message: "Select workflow:",
			Options: workflowOptions,
		}
		if err := survey.AskOne(promptWorkflow, &selectedWorkflow); err != nil {
			return err
		}

		workflow := workflowMap[selectedWorkflow]
		workflowID = workflow.GetID()

		// Ask for branch
		promptBranch := &survey.Input{
			Message: "Enter branch name (leave empty for default):",
		}
		if err := survey.AskOne(promptBranch, &branch); err != nil {
			return err
		}

		promptVersionBuilder := &survey.Input{
			Message: "Actions version builder (optional):",
			Help:    "Go template. e.g. v{{.RunNumber}}, {{.Branch}}-{{.RunNumber}}, v{{add .RunNumber 1000}}, {{.CreatedAt}}.{{pad .RunNumber 4}}",
			Default: "{{.Branch}}-{{.ShortSHA}}",
		}
		if err := survey.AskOne(promptVersionBuilder, &actionsVersionBuilder); err != nil {
			return err
		}
		actionsVersionBuilder = strings.TrimSpace(actionsVersionBuilder)

		// Get recent successful runs
		fmt.Printf("Fetching recent successful runs...\n")
		runs, _, err := client.Actions.ListWorkflowRunsByID(ctx, owner, repo, workflowID, &github.ListWorkflowRunsOptions{
			Branch: branch,
			Status: "success",
			ListOptions: github.ListOptions{
				PerPage: 5,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to fetch workflow runs: %w", err)
		}

		if len(runs.WorkflowRuns) == 0 {
			return fmt.Errorf("no successful workflow runs found")
		}

		// Use the most recent run
		run := runs.WorkflowRuns[0]
		fmt.Printf("Using run: %s (commit: %s)\n", run.GetName(), run.GetHeadSHA()[:7])

		// Get artifacts from the run
		artifacts, _, err := client.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, run.GetID(), nil)
		if err != nil {
			return fmt.Errorf("failed to list artifacts: %w", err)
		}

		if len(artifacts.Artifacts) == 0 {
			return fmt.Errorf("no artifacts found in the workflow run")
		}

		var artifactOptions []string
		artifactMap := make(map[string]*github.Artifact)
		for _, artifact := range artifacts.Artifacts {
			name := artifact.GetName()
			artifactOptions = append(artifactOptions, name)
			artifactMap[name] = artifact
		}

		promptArtifacts := &survey.MultiSelect{
			Message: "Select artifacts to include in the rule:",
			Options: artifactOptions,
		}
		if err := survey.AskOne(promptArtifacts, &selectedAssets); err != nil {
			return err
		}

		promptModID := &survey.Input{
			Message: "Enter Mod ID:",
			Default: repo,
		}
		if err := survey.AskOne(promptModID, &modID); err != nil {
			return err
		}

		// Process actions artifacts
		for _, artifactName := range selectedAssets {
			artifact := artifactMap[artifactName]
			fmt.Printf("\nConfiguring artifact: %s\n", artifact.GetName())

			fRule := FileRule{
				TargetPlatform: "any",
				ContentType:    string(model.ContentTypeArchive),
				Source:         "actions",
				WorkflowID:     fmt.Sprintf("%d", workflowID),
				Branch:         branch,
			}

			// Ask for artifact name pattern
			var artifactPattern string
			promptPattern := &survey.Input{
				Message: "Enter artifact name pattern (regex):",
				Default: artifact.GetName(),
			}
			if err := survey.AskOne(promptPattern, &artifactPattern); err != nil {
				return err
			}
			fRule.ArtifactName = artifactPattern

			// Ask for file pattern within artifact
			var filePattern string
			promptFilePattern := &survey.Input{
				Message: "Enter file pattern within artifact (optional, regex):",
			}
			if err := survey.AskOne(promptFilePattern, &filePattern); err != nil {
				return err
			}
			fRule.FilePattern = filePattern

			var contentType string
			promptContentType := &survey.Select{
				Message: "Select content type:",
				Options: []string{"Archive", "Binary", "PluginDll"},
				Default: "Archive",
			}
			if err := survey.AskOne(promptContentType, &contentType); err != nil {
				return err
			}
			fRule.ContentType = contentTypeMap(contentType)

			var extractPath string
			promptExtract := &survey.Input{
				Message: "Extract path (optional):",
			}
			if err := survey.AskOne(promptExtract, &extractPath); err != nil {
				return err
			}
			fRule.ExtractPath = extractPath

			var targetPlatform string
			promptPlatform := &survey.Select{
				Message: "Target platform:",
				Options: []string{"any", "windows", "linux", "darwin"},
				Default: "any",
			}
			if err := survey.AskOne(promptPlatform, &targetPlatform); err != nil {
				return err
			}
			fRule.TargetPlatform = targetPlatform

			fileRules = append(fileRules, fRule)
		}
	}

	// Ask for custom files
	for {
		addCustom := false
		promptCustom := &survey.Confirm{
			Message: "Add a custom file (URL)?",
			Default: false,
		}
		if err := survey.AskOne(promptCustom, &addCustom); err != nil {
			return err
		}
		if !addCustom {
			break
		}

		var fRule FileRule
		fRule.TargetPlatform = "any"

		promptUrl := &survey.Input{Message: "URL:"}
		if err := survey.AskOne(promptUrl, &fRule.From); err != nil {
			return err
		}

		promptContentType := &survey.Select{
			Message: "Content Type:",
			Options: []string{string(model.ContentTypeArchive), string(model.ContentTypeBinary), string(model.ContentTypePluginDll)},
			Default: string(model.ContentTypeArchive),
		}
		if err := survey.AskOne(promptContentType, &fRule.ContentType); err != nil {
			return err
		}

		promptExtract := &survey.Input{Message: "Extract Path (optional):"}
		if err := survey.AskOne(promptExtract, &fRule.ExtractPath); err != nil {
			return err
		}

		fileRules = append(fileRules, fRule)
	}

	var dependencies []Dependency
	for {
		addDep := false
		promptDep := &survey.Confirm{
			Message: "Add a dependency?",
			Default: false,
		}
		if err := survey.AskOne(promptDep, &addDep); err != nil {
			return err
		}
		if !addDep {
			break
		}

		var dep Dependency
		promptMod := &survey.Input{Message: "Dependency Mod ID:"}
		if err := survey.AskOne(promptMod, &dep.ModID); err != nil {
			return err
		}

		promptVer := &survey.Input{
			Message: "Version ID (use 'any' for any version):",
			Default: "any",
		}
		if err := survey.AskOne(promptVer, &dep.VersionID); err != nil {
			return err
		}

		promptType := &survey.Select{
			Message: "Dependency Type:",
			Options: []string{"required", "optional", "incompatible"},
			Default: "required",
		}
		if err := survey.AskOne(promptType, &dep.DependencyType); err != nil {
			return err
		}

		dependencies = append(dependencies, dep)
	}

	var features []Feature
	for {
		addFeature := false
		promptFeature := &survey.Confirm{
			Message: "Add a feature?",
			Default: false,
		}
		if err := survey.AskOne(promptFeature, &addFeature); err != nil {
			return err
		}
		if !addFeature {
			break
		}

		var feature Feature
		promptName := &survey.Input{
			Message: "Feature Name:",
			Default: "direct_join",
		}
		if err := survey.AskOne(promptName, &feature.Name); err != nil {
			return err
		}
		feature.Name = strings.TrimSpace(feature.Name)
		if feature.Name == "" {
			return fmt.Errorf("feature name cannot be empty")
		}

		var valueType string
		promptValueType := &survey.Select{
			Message: "Feature Value Type:",
			Options: []string{"bool", "string", "number"},
			Default: "bool",
		}
		if err := survey.AskOne(promptValueType, &valueType); err != nil {
			return err
		}

		switch valueType {
		case "bool":
			var b bool
			promptBool := &survey.Confirm{
				Message: "Feature Value:",
				Default: true,
			}
			if err := survey.AskOne(promptBool, &b); err != nil {
				return err
			}
			feature.Value = b
		case "string":
			var s string
			promptString := &survey.Input{
				Message: "Feature Value:",
			}
			if err := survey.AskOne(promptString, &s); err != nil {
				return err
			}
			feature.Value = s
		case "number":
			var raw string
			promptNumber := &survey.Input{
				Message: "Feature Value (number):",
				Default: "1",
			}
			if err := survey.AskOne(promptNumber, &raw); err != nil {
				return err
			}
			raw = strings.TrimSpace(raw)
			if strings.Contains(raw, ".") {
				var f float64
				if _, err := fmt.Sscanf(raw, "%f", &f); err != nil {
					return fmt.Errorf("invalid number value: %s", raw)
				}
				feature.Value = f
			} else {
				var i int64
				if _, err := fmt.Sscanf(raw, "%d", &i); err != nil {
					return fmt.Errorf("invalid number value: %s", raw)
				}
				feature.Value = i
			}
		}

		features = append(features, feature)
	}

	rule := Rule{
		ModID:                 modID,
		GithubRepo:            repoInput,
		ActionsVersionBuilder: actionsVersionBuilder,
		Dependencies:          dependencies,
		Features:              features,
		Files:                 fileRules,
	}

	defaultPath := filepath.Join("rules", modID+".rule.json")
	var savePath string
	promptSave := &survey.Input{
		Message: "Save rule to file:",
		Default: defaultPath,
	}
	if err := survey.AskOne(promptSave, &savePath); err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(rule, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return err
	}

	fmt.Printf("Rule saved to %s\n", savePath)
	return nil
}
