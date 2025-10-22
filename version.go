package main

import _ "embed"

//go:embed version
var version string

func checkUpdate() error {
	return nil
}
