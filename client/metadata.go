//go:build windows

package main

import (
	_ "embed"
	"fmt"
	"runtime/debug"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

//go:embed icon.png
var icon []byte

var version string

func init() {
	info, ok := debug.ReadBuildInfo()
	if ok {
		fmt.Println(info)
		version = info.Main.Version
		vscIdx := -1
		for i, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				vscIdx = i
				break
			}
		}
		if vscIdx != -1 {
			version += " (" + info.Settings[vscIdx].Value[:min(7, len(info.Settings[vscIdx].Value))] + ")"
		}
	}

	app.SetMetadata(fyne.AppMetadata{
		ID:      "com.github.ikafly.au_mod_installer",
		Name:    "AU Mod Installer",
		Version: version,
		Build:   1,
		Icon:    fyne.NewStaticResource("icon.png", icon),
	})
}
