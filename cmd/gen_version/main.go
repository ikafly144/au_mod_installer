package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	_ "unsafe"

	_ "golang.org/x/mod/semver"
)

var jsonTemplate = `{
	"FixedFileInfo":
	{
		"FileVersion": {
			"Major": %d,
			"Minor": %d,
			"Patch": %d,
			"Build": %d
		},
		"ProductVersion": {
			"Major": %d,
			"Minor": %d,
			"Patch": %d,
			"Build": %d
		},
		"FileFlagsMask": "3f",
		"FileFlags ": "00",
		"FileOS": "040004",
		"FileType": "01",
		"FileSubType": "00"
	},
	"StringFileInfo":
	{
		"Comments": "",
		"CompanyName": "ikafly144",
		"FileDescription": "Mod of Us - Among Us Mod Manager",
		"FileVersion": "%s",
		"InternalName": "",
		"LegalCopyright": "Copyright (C) 2026 ikafly144.",
		"LegalTrademarks": "",
		"OriginalFilename": "MOD-OF-US.EXE",
		"PrivateBuild": "",
		"ProductName": "Mod of Us - Among Us Mod Manager",
		"ProductVersion": "%s",
		"SpecialBuild": ""
	},
	"VarFileInfo":
	{
		"Translation": {
			"LangID": "0411",
			"CharsetID": "04E4"
		}
	}
}
`

//nolint:unused
type parsed struct {
	major      string
	minor      string
	patch      string
	short      string
	prerelease string
	build      string
}

//go:linkname parse golang.org/x/mod/semver.parse
func parse(v string) (parsed, bool)

func versionStrToNum(versionString string) ([]int, error) {
	vn := make([]int, 4)
	version, ok := parse(versionString)
	if !ok {
		return nil, fmt.Errorf("%s: invalid semantic version", versionString)
	}
	var err error
	if vn[0], err = strconv.Atoi(version.major); err != nil {
		return nil, fmt.Errorf("%s: invalid major version", versionString)
	}
	if vn[1], err = strconv.Atoi(version.minor); err != nil {
		return nil, fmt.Errorf("%s: invalid minor version", versionString)
	}
	if vn[2], err = strconv.Atoi(version.patch); err != nil {
		return nil, fmt.Errorf("%s: invalid patch version", versionString)
	}
	// Build number is unused in semver, so we set it to 0
	vn[3] = 0
	return vn[:], nil
}

func getVersionData() (string, []int, error) {
	bin, err := exec.Command("git", "describe", "--tags").Output()
	if err != nil {
		return "", nil, fmt.Errorf("Could not get version string from git (%w)", err)
	}
	str := strings.TrimSpace(string(bin))
	num, err := versionStrToNum(str)
	return str, num, err
}

func main() {
	fileVerStr, fileVerNum, err := getVersionData()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	fmt.Printf(jsonTemplate,
		fileVerNum[0],
		fileVerNum[1],
		fileVerNum[2],
		fileVerNum[3],
		fileVerNum[0],
		fileVerNum[1],
		fileVerNum[2],
		fileVerNum[3],
		fileVerStr,
		fileVerStr)
}
