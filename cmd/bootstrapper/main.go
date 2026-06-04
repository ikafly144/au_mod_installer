//go:build bootstrapper

package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed mod-of-us.msi
var msiData []byte

func main() {
	tempDir := os.TempDir()
	msiPath := filepath.Join(tempDir, "mod-of-us-installer.msi")

	err := os.WriteFile(msiPath, msiData, 0644)
	if err != nil {
		fmt.Printf("Failed to extract installer: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Starting installer...")
	cmd := exec.Command("msiexec", "/i", msiPath)
	err = cmd.Start()
	if err != nil {
		fmt.Printf("Failed to start installer: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Installer launched successfully.")
	// We don't wait for the installer to finish as it might take some time and
	// msiexec runs as a separate process. The temp file will remain until cleaned by OS
	// or we could try to delete it after a delay, but msiexec needs it while running.
}
