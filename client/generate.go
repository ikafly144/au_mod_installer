package main

//go:generate cmd /c go run ../cmd/gen_version/main.go > versioninfo.json && go run github.com/hymkor/goversioninfo/cmd/goversioninfo@latest -icon=icon.ico -o mod-of-us.syso versioninfo.json && del versioninfo.json
