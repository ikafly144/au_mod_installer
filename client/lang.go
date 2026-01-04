package main

import (
	"embed"

	"fyne.io/fyne/v2/lang"
)

//go:embed locale/*
var locale embed.FS

func init() {
	if err := lang.AddTranslationsFS(locale, "locale"); err != nil {
		panic(err)
	}
}
