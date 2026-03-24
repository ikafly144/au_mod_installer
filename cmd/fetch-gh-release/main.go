package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

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
}

type Rule struct {
	ModID        string       `json:"mod_id"`
	GithubRepo   string       `json:"github_repo"`
	Dependencies []Dependency `json:"dependencies"`
	Files        []FileRule   `json:"files"`
}

type Output struct {
	ModID        string   `json:"mod_id"`
	VersionID    string   `json:"version_id"`
	Files        []string `json:"files"`
	Dependencies []string `json:"dependencies"`
}

func main() {
	ruleFile := flag.String("rule", "", "Path to the rule file")
	flag.Parse()

	if *ruleFile == "" {
		fmt.Fprintln(os.Stderr, "Usage: fetch-gh-release -rule <path-to-rule.json>")
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

	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch latest release from %s: %v\n", rule.GithubRepo, err)
		os.Exit(1)
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

	outJson, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(outJson))
}
