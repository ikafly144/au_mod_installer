package main

//go:generate go run ../cmd/gen_syso/ -o version.syso -icon icon.ico -arch amd64
//go:generate go run ../cmd/gen_licenses/ -target github.com/ikafly144/au_mod_installer/client -output ui/tab/settings/licenses.json
