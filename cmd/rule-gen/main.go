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
}

type Feature struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type Rule struct {
	ModID        string       `json:"mod_id"`
	GithubRepo   string       `json:"github_repo"`
	Dependencies []Dependency `json:"dependencies"`
	Features     []Feature    `json:"features,omitempty"`
	Files        []FileRule   `json:"files"`
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

	fmt.Printf("Fetching latest release for %s/%s...\n", owner, repo)
	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to fetch release: %w", err)
	}

	fmt.Printf("Found release: %s\n", release.GetTagName())

	var modID string
	promptModID := &survey.Input{
		Message: "Enter Mod ID:",
		Default: repo,
	}
	if err := survey.AskOne(promptModID, &modID); err != nil {
		return err
	}

	var fileRules []FileRule
	var assetOptions []string
	assetMap := make(map[string]*github.ReleaseAsset)

	for _, asset := range release.Assets {
		name := asset.GetName()
		assetOptions = append(assetOptions, name)
		assetMap[name] = asset
	}

	var selectedAssets []string
	promptAssets := &survey.MultiSelect{
		Message: "Select assets to include in the rule:",
		Options: assetOptions,
	}
	if err := survey.AskOne(promptAssets, &selectedAssets); err != nil {
		return err
	}

	for _, assetName := range selectedAssets {
		fmt.Printf("\nConfiguring asset: %s\n", assetName)

		var ruleType string
		promptType := &survey.Select{
			Message: "Use explicit URL or Artifact Regex?",
			Options: []string{"Artifact Regex (Recommended)", "Explicit URL"},
			Default: "Artifact Regex (Recommended)",
		}
		if err := survey.AskOne(promptType, &ruleType); err != nil {
			return err
		}

		fRule := FileRule{
			TargetPlatform: "any",
			ContentType:    string(model.ContentTypeArchive),
		}

		if ruleType == "Artifact Regex (Recommended)" {
			// Escape special characters for regex
			escaped := strings.ReplaceAll(assetName, ".", "\\.")
			// Suggest replacing version numbers with .*
			version := strings.TrimPrefix(release.GetTagName(), "v")
			if version != "" {
				escaped = strings.ReplaceAll(escaped, version, ".*")
			}

			promptRegex := &survey.Input{
				Message: "Artifact Regex:",
				Default: escaped,
			}
			if err := survey.AskOne(promptRegex, &fRule.Artifact); err != nil {
				return err
			}
		} else {
			fRule.From = assetMap[assetName].GetBrowserDownloadURL()
		}

		promptContentType := &survey.Select{
			Message: "Content Type:",
			Options: []string{string(model.ContentTypeArchive), string(model.ContentTypeBinary), string(model.ContentTypePluginDll)},
			Default: string(model.ContentTypeArchive),
		}
		if err := survey.AskOne(promptContentType, &fRule.ContentType); err != nil {
			return err
		}

		promptExtract := &survey.Input{
			Message: "Extract Path (optional, leave empty for auto):",
		}
		if err := survey.AskOne(promptExtract, &fRule.ExtractPath); err != nil {
			return err
		}

		promptPlatform := &survey.Select{
			Message: "Target Platform:",
			Options: []string{"any", "windows", "linux", "darwin"},
			Default: "any",
		}
		if err := survey.AskOne(promptPlatform, &fRule.TargetPlatform); err != nil {
			return err
		}

		fileRules = append(fileRules, fRule)
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
		ModID:        modID,
		GithubRepo:   repoInput,
		Dependencies: dependencies,
		Features:     features,
		Files:        fileRules,
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

	data, err := json.Marshal(rule)
	if err != nil {
		return err
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return err
	}

	fmt.Printf("Rule saved to %s\n", savePath)
	return nil
}
