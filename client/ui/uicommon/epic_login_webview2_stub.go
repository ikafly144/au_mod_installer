//go:build !windows

package uicommon

func startEpicWebView2Login(_ string) (<-chan string, <-chan error, func()) {
	return nil, nil, func() {}
}
