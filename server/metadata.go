package main

import (
	"fmt"
	"runtime/debug"
)

var (
	version  string
	revision string
)

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
			revision = info.Settings[vscIdx].Value
		} else {
			revision = "unknown"
		}
	} else {
		version = "unknown"
		revision = "unknown"
	}
}
