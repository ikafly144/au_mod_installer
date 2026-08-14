//go:build !windows

package uicommon

// IsAutoStartEnabled checks if the application is set to launch on startup.
func IsAutoStartEnabled() bool {
	return false
}

// SetAutoStartEnabled enables or disables launching the application on startup.
func SetAutoStartEnabled(enabled bool) error {
	return nil
}

// SyncAutoStart ensures the startup entry matches the preference.
func SyncAutoStart(enabled bool) {
}
