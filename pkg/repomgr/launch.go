package repomgr

// Launch launches R.E.P.O. with Doorstop support.
func Launch(launcherType LauncherType, gameDir string, dllDir string, onStarted func(pid int) error) error {
	return LaunchRepo(gameDir, dllDir, onStarted)
}
