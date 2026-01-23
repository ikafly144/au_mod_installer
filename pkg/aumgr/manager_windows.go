//go:build windows

package aumgr

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func GetAmongUsDir() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, "SOFTWARE\\Classes\\amongus\\shell\\open\\command", registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()

	val, _, err := key.GetStringValue("")
	if err != nil {
		return "", err
	}
	val = strings.Trim(strings.TrimSpace(val[0:len(val)-4]), "\"")
	val, ok := strings.CutSuffix(val, "Among Us_Data\\Resources\\AmongUsHelper.exe")
	if !ok {
		return "", fmt.Errorf("among Us Helper is not supported %s", val)
	}
	return val, nil
}
