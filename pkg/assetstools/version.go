package assetstools

import (
	"encoding/binary"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func ReadPlayerSettingsBundleVersion(globalGameManagersPath string) (string, error) {
	f, err := os.Open(globalGameManagersPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	reader := NewAssetsFileReader(f)
	assetsFile, err := ReadAssetsFile(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read assets file: %w", err)
	}

	playerSettings, err := findSingleAssetByClassID(assetsFile, int32(ClassIDPlayerSettings))
	if err != nil {
		return "", err
	}

	if assetsFile.Metadata.TypeTreeEnabled {
		version, err := extractBundleVersionWithTypeTree(reader, assetsFile, playerSettings)
		if err == nil {
			return version, nil
		}
	}

	data, err := readAssetData(reader, assetsFile, playerSettings)
	if err != nil {
		return "", err
	}
	version, err := extractBundleVersionFromSerializedData(data)
	if err != nil {
		return "", fmt.Errorf("failed to extract bundleVersion from PlayerSettings: %w", err)
	}
	return version, nil
}

func findSingleAssetByClassID(file *AssetsFile, classID int32) (*AssetFileInfo, error) {
	var found *AssetFileInfo
	for i := range file.Metadata.AssetInfos {
		info := &file.Metadata.AssetInfos[i]
		if info.TypeID != classID {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("multiple assets found for class id %d", classID)
		}
		found = info
	}
	if found == nil {
		return nil, fmt.Errorf("asset with class id %d not found", classID)
	}
	return found, nil
}

func extractBundleVersionWithTypeTree(reader *AssetsFileReader, file *AssetsFile, info *AssetFileInfo) (string, error) {
	typeTreeType := file.Metadata.FindTypeTreeTypeByID(int32(ClassIDPlayerSettings), -1)
	if typeTreeType == nil {
		return "", fmt.Errorf("type tree for PlayerSettings not found")
	}
	template, err := NewTemplateFieldFromTypeTree(typeTreeType)
	if err != nil {
		return "", err
	}

	value, err := template.MakeValueAt(reader, info.GetAbsoluteByteOffset(file))
	if err != nil {
		return "", fmt.Errorf("failed to decode PlayerSettings value: %w", err)
	}

	bundleVersionField := value.Find("bundleVersion")
	if bundleVersionField == nil {
		return "", fmt.Errorf("bundleVersion field not found")
	}
	bundleVersion, ok := bundleVersionField.AsString()
	if !ok {
		return "", fmt.Errorf("bundleVersion is not a string")
	}
	if bundleVersion == "" {
		return "", fmt.Errorf("bundleVersion is empty")
	}
	return bundleVersion, nil
}

func readAssetData(reader *AssetsFileReader, file *AssetsFile, info *AssetFileInfo) ([]byte, error) {
	if uint64(info.ByteSize) > uint64(int(^uint(0)>>1)) {
		return nil, fmt.Errorf("asset too large: %d", info.ByteSize)
	}
	if err := reader.SeekAt(info.GetAbsoluteByteOffset(file)); err != nil {
		return nil, err
	}
	return reader.ReadBytes(int(info.ByteSize))
}

var bundleVersionPattern = regexp.MustCompile(`\b\d{4}\.\d+\.\d+\b`)

func extractBundleVersionFromSerializedData(data []byte) (string, error) {
	candidates := make([]string, 0)
	seen := map[string]struct{}{}

	for i := 0; i+4 <= len(data); i++ {
		n := int(int32(binary.LittleEndian.Uint32(data[i : i+4])))
		if n < 6 || n > 128 || i+4+n > len(data) {
			continue
		}
		value := string(data[i+4 : i+4+n])
		if !isPrintableASCII(value) || !bundleVersionPattern.MatchString(value) {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		candidates = append(candidates, value)
	}

	if len(candidates) == 0 {
		for i := 0; i < len(data); {
			if data[i] < 32 || data[i] > 126 {
				i++
				continue
			}
			start := i
			for i < len(data) && data[i] >= 32 && data[i] <= 126 {
				i++
			}
			value := string(data[start:i])
			if bundleVersionPattern.MatchString(value) {
				if _, ok := seen[value]; !ok {
					seen[value] = struct{}{}
					candidates = append(candidates, value)
				}
			}
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("version-like string was not found")
	}
	return pickLatestVersion(candidates)
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 32 || s[i] > 126 {
			return false
		}
	}
	return true
}

func pickLatestVersion(candidates []string) (string, error) {
	best := ""
	var bestCore [3]int
	for _, candidate := range candidates {
		core, ok := parseVersionCore(candidate)
		if !ok {
			continue
		}
		if best == "" || compareVersionCore(core, bestCore) > 0 {
			best = candidate
			bestCore = core
		}
	}
	if best == "" {
		return "", fmt.Errorf("no valid version candidate found")
	}
	return best, nil
}

func parseVersionCore(version string) ([3]int, bool) {
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return [3]int{}, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return [3]int{}, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return [3]int{}, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return [3]int{}, false
	}
	return [3]int{major, minor, patch}, true
}

func compareVersionCore(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] > b[i] {
			return 1
		}
		if a[i] < b[i] {
			return -1
		}
	}
	return 0
}
