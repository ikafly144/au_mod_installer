package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type ThirdPartyLicense struct {
	Name        string `json:"name"`
	LicenseName string `json:"license_name"`
	LicenseURL  string `json:"license_url"`
	LicenseText string `json:"license_text"`
}

type ProjectLicense struct {
	LicenseURL  string `json:"license_url"`
	LicenseName string `json:"license_name"`
	LicenseText string `json:"license_text"`
}

type LicensesDocument struct {
	Project    ProjectLicense      `json:"project"`
	ThirdParty []ThirdPartyLicense `json:"third_party"`
}

func main() {
	target := flag.String("target", "", "Go package/module path to analyze")
	output := flag.String("output", "", "Output JSON path")
	goos := flag.String("goos", "", "GOOS value for go-licenses (optional)")
	cgoEnabled := flag.Bool("cgo", false, "Whether to enable CGO for go-licenses (optional)")
	flag.Parse()

	if strings.TrimSpace(*target) == "" {
		fmt.Fprintln(os.Stderr, "target is required")
		os.Exit(1)
	}
	if strings.TrimSpace(*output) == "" {
		fmt.Fprintln(os.Stderr, "output is required")
		os.Exit(1)
	}

	if err := ensureGoLicenses(); err != nil {
		fmt.Fprintln(os.Stderr, "failed to ensure go-licenses:", err)
		os.Exit(1)
	}

	modulePath, err := currentModulePath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to resolve current module path:", err)
		os.Exit(1)
	}
	moduleDir, err := currentModuleDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to resolve current module directory:", err)
		os.Exit(1)
	}

	env := []string{
		fmt.Sprintf("GOOS=%s", *goos),
	}
	if *cgoEnabled {
		env = append(env, "CGO_ENABLED=1")
	}

	reportOutput, err := runCmd(env, "go-licenses", "report", *target)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to generate go-licenses report:", err)
		os.Exit(1)
	}

	reportRows, err := parseReportCSV(reportOutput)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to parse go-licenses report:", err)
		os.Exit(1)
	}

	tempRoot, err := os.MkdirTemp("", "go-licenses-save-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create temp directory:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempRoot)

	savePath := filepath.Join(tempRoot, "licenses")
	if _, err := runCmd(env, "go-licenses", "save", *target, "--save_path", savePath); err != nil {
		fmt.Fprintln(os.Stderr, "failed to collect license texts:", err)
		os.Exit(1)
	}

	licenses := make([]ThirdPartyLicense, 0, len(reportRows))
	for _, row := range reportRows {
		if len(row) < 3 {
			continue
		}
		name := strings.TrimSpace(row[0])
		licenseURL := strings.TrimSpace(strings.ReplaceAll(row[1], "\\", "/"))
		licenseName := strings.TrimSpace(row[2])
		if name == "" || licenseURL == "" || licenseName == "" {
			continue
		}
		if name == modulePath || strings.HasPrefix(name, modulePath+"/") {
			continue
		}

		licensePath, err := findLicenseFile(savePath, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to find license file for %s: %v\n", name, err)
			os.Exit(1)
		}
		licenseText, err := os.ReadFile(licensePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read license text for %s: %v\n", name, err)
			os.Exit(1)
		}

		licenses = append(licenses, ThirdPartyLicense{
			Name:        name,
			LicenseName: licenseName,
			LicenseURL:  licenseURL,
			LicenseText: strings.TrimSpace(string(licenseText)),
		})
	}

	sort.Slice(licenses, func(i, j int) bool {
		return licenses[i].Name < licenses[j].Name
	})

	projectLicenseTextBytes, err := os.ReadFile(filepath.Join(moduleDir, "LICENSE"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to read project LICENSE:", err)
		os.Exit(1)
	}
	document := LicensesDocument{
		Project: ProjectLicense{
			LicenseURL:  "https://github.com/ikafly144/au_mod_installer/blob/master/LICENSE",
			LicenseName: "GPL-3.0",
			LicenseText: strings.TrimSpace(string(projectLicenseTextBytes)),
		},
		ThirdParty: licenses,
	}

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "failed to create output directory:", err)
		os.Exit(1)
	}

	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to encode JSON:", err)
		os.Exit(1)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(*output, encoded, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "failed to write output file:", err)
		os.Exit(1)
	}
}

func parseReportCSV(content string) ([][]string, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = -1
	return reader.ReadAll()
}

func runCmd(env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), env...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return "", fmt.Errorf("%w: %s", err, stderrText)
		}
		return "", err
	}
	return stdout.String(), nil
}

func currentModulePath() (string, error) {
	output, err := runCmd(nil, "go", "list", "-m", "-f", "{{.Path}}")
	if err != nil {
		return "", err
	}
	modulePath := strings.TrimSpace(output)
	if modulePath == "" {
		return "", fmt.Errorf("empty module path")
	}
	return modulePath, nil
}

func currentModuleDir() (string, error) {
	output, err := runCmd(nil, "go", "list", "-m", "-f", "{{.Dir}}")
	if err != nil {
		return "", err
	}
	moduleDir := strings.TrimSpace(output)
	if moduleDir == "" {
		return "", fmt.Errorf("empty module directory")
	}
	return moduleDir, nil
}

func ensureGoLicenses() error {
	if _, err := exec.LookPath("go-licenses"); err == nil {
		return nil
	}

	fmt.Fprintln(os.Stderr, "go-licenses not found, installing...")
	cmd := exec.Command("go", "install", "github.com/google/go-licenses@latest")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install go-licenses: %w", err)
	}
	fmt.Fprintln(os.Stderr, "go-licenses installed successfully")
	return nil
}

func findLicenseFile(baseDir string, packageName string) (string, error) {
	p := filepath.Join(append([]string{baseDir}, strings.Split(packageName, "/")...)...)
	info, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return p, nil
	}

	entries, err := os.ReadDir(p)
	if err != nil {
		return "", err
	}
	candidates := []string{"LICENSE", "LICENSE.txt", "LICENSE.md", "COPYING", "COPYING.txt"}
	for _, candidate := range candidates {
		full := filepath.Join(p, candidate)
		if _, err := os.Stat(full); err == nil {
			return full, nil
		}
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		nameUpper := strings.ToUpper(entry.Name())
		if strings.Contains(nameUpper, "LICENSE") || strings.Contains(nameUpper, "COPYING") {
			return filepath.Join(p, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("license file not found in %s", p)
}
