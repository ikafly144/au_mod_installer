//go:build windows

package main

import (
	"bufio"
	"context"
	"flag"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"
	"github.com/Microsoft/go-winio"
	"github.com/nightlyone/lockfile"
	"github.com/sqweek/dialog"
	"github.com/zzl/go-win32api/win32"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/client/ui"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/ikafly144/au_mod_installer/common/versioning"
)

var DefaultServer = "https://modofus.sabafly.net/api/v1"
var pipeName = `\\.\pipe\au_mod_installer_ipc`

func main() {
	sharedURI := ""
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "mod-of-us://") {
			sharedURI = arg
			break
		}
	}

	lockPath := filepath.Join(os.Getenv("PROGRAMDATA"), "au_mod_installer.lock")
	lock, err := lockfile.New(lockPath)
	if err != nil {
		slog.Error("Failed to create lockfile", "error", err)
		os.Exit(1)
	}
	err = lock.TryLock()
	if err != nil {
		slog.Error("Another instance is already running", "error", err)

		// Try to send URI to the existing instance via IPC
		if sharedURI != "" {
			conn, err := winio.DialPipe(pipeName, nil)
			if err == nil {
				defer conn.Close()
				_, _ = conn.Write([]byte(sharedURI + "\n"))
				slog.Info("Sent shared URI to existing instance", "uri", sharedURI)
			}
		}

		owner, err := lock.GetOwner()
		if current, err1 := os.FindProcess(os.Getpid()); err == nil && err1 == nil {
			slog.Info("Lockfile owned by", "pid", owner.Pid)
			slog.Info("Current process pid", "pid", current.Pid)
			if owner.Pid == current.Pid {
				slog.Info("Lockfile owned by current process, unlocking")
				_ = os.Remove(lockPath)
			} else {
				found := false
				if err := windows.EnumWindows(syscall.NewCallback(func(hwnd windows.HWND, lparam uintptr) int {
					var pid uint32
					tid, err := windows.GetWindowThreadProcessId(hwnd, &pid)
					if err != nil || tid == 0 {
						return 1
					}
					if int(pid) == owner.Pid {
						var classNamePtr [256]uint16
						if _, err := windows.GetClassName(hwnd, &classNamePtr[0], int32(len(classNamePtr))); err != nil {
							slog.Error("Failed to get window class name", "error", err)
						}
						className := syscall.UTF16ToString(classNamePtr[:])
						if !strings.Contains(className, "GLFW") && !strings.Contains(className, "NVOpenGL") {
							return 1
						}
						slog.Info("Found window of existing process, bringing to foreground", "hwnd", hwnd, "pid", pid, "class", className)
						win32.FlashWindowEx(&win32.FLASHWINFO{
							CbSize:    uint32(unsafe.Sizeof(win32.FLASHWINFO{})),
							Hwnd:      win32.HWND(hwnd),
							DwFlags:   win32.FLASHW_TRAY | win32.FLASHW_TIMERNOFG,
							UCount:    5,
							DwTimeout: 0,
						})
						found = win32.SetForegroundWindow(win32.HWND(hwnd)) != win32.FALSE
						return 1
					}
					return 1
				}), nil); err != nil {
					slog.Error("Failed to enumerate windows", "error", err)
				}
				if found {
					slog.Info("Brought existing instance to foreground, exiting")
				} else {
					if err := owner.Kill(); err != nil {
						slog.Error("Failed to kill existing process", "error", err)
					}
					(&dialog.MsgBuilder{Msg: lang.LocalizeKey("app.error.already_running", "Another instance of Mod of Us was already running and has been forced to close. Please restart the application.")}).Title(lang.LocalizeKey("app.error", "Error")).Error()
				}
			}
		}
		_ = lock.Unlock()
		os.Exit(1)
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			slog.Error("Failed to unlock lockfile", "error", err)
		}
	}()
	mainErr := realMain(sharedURI)
	if mainErr != nil {
		os.Exit(1)
	}
}

func realMain(sharedURI string) error {
	var (
		localMode string
		server    string
		offline   bool
	)

	a := app.New()

	flag.StringVar(&localMode, "local", "", "Path to local mods.json file for local mode")
	flag.StringVar(&server, "server", "", "URL of the mod server")
	flag.BoolVar(&offline, "offline", false, "Run in offline mode (only uninstallation and management of installed mods are available)")
	flag.Parse()

	if server == "" {
		server = a.Preferences().StringWithFallback("api_server", DefaultServer)
	}

	if err := registerScheme(); err != nil {
		slog.Error("Failed to register scheme", "error", err)
	}

	branch := versioning.BranchFromString(a.Preferences().StringWithFallback("core.update_branch", "stable"))

	tag, err := versioning.CheckForUpdates(context.Background(), branch, version)
	if err != nil {
		slog.Error("Failed to check for updates", "error", err)
	} else if tag != "" {
		slog.Info("Update available", "version", tag)
		yes := (&dialog.MsgBuilder{Msg: lang.LocalizeKey("update.available", "New version \"{{.Version}}\" is available. Click 'Yes' to update.", map[string]any{"Version": tag})}).Title(lang.LocalizeKey("update.title", "Update Available")).YesNo()
		if yes {
			slog.Info("Updating to new version", "version", tag)
			if err := versioning.Update(context.Background(), tag); err != nil {
				slog.Error("Failed to update", "error", err)
				(&dialog.MsgBuilder{Msg: lang.LocalizeKey("update.failed", "Update failed: {{.Error}}", map[string]any{"Error": err.Error()})}).Title(lang.LocalizeKey("app.error", "Error")).Error()
				return err
			}
			execCmd := exec.Command(os.Args[0], os.Args[1:]...)
			return execCmd.Start()
		}
	} else {
		slog.Info("No updates available")
	}

	w := a.NewWindow(lang.LocalizeKey("app.name", "Mod of Us"))

	var client rest.Client
	if localMode != "" {
		slog.Info("Running in local mode", "path", localMode)
		f, err := rest.NewFileClient(localMode)
		if err != nil {
			slog.Error("Failed to create local file client", "error", err)
			dialog.Message(lang.LocalizeKey("error.local_client_creation_failed", "Failed to create local file client: %s"), err.Error()).Title(lang.LocalizeKey("app.error", "Error")).Error()
			return err
		}
		if err := f.LoadData(); err != nil {
			slog.Error("Failed to load data from local file", "error", err)
			dialog.Message(lang.LocalizeKey("error.local_data_load_failed", "Failed to load data from local file: %s"), err.Error()).Title(lang.LocalizeKey("app.error", "Error")).Error()
			return err
		}
		client = f
	} else if offline {
		slog.Info("Running in offline mode")
		client = rest.NewOfflineClient()
	} else {
		slog.Info("Running in server mode", "server", server)
		client = rest.NewClient(server)

		if _, err := client.GetHealthStatus(); err != nil {
			slog.Error("Failed to connect to server", "error", err)
			yes := (&dialog.MsgBuilder{Msg: lang.LocalizeKey("error.server_connection_failed_offline_prompt", "Failed to connect to server: {{.Error}}\nDo you want to continue in offline mode?\n(Only uninstallation and management of installed mods are available)", map[string]any{"Error": err})}).Title(lang.LocalizeKey("error.connection_error", "Connection Error")).YesNo()
			if yes {
				slog.Info("Continuing in offline mode")
				client = rest.NewOfflineClient()
			} else {
				return err
			}
		}
	}

	if err := ui.Main(w, version, sharedURI,
		ui.WithStateOptions(
			uicommon.WithRestClient(client),
		),
		ui.WithStateInit(func(s *uicommon.State) {
			go startIPCListener(s)
		}),
	); err != nil {
		slog.Error("Failed to initialize UI", "error", err)
		dialog.Message(lang.LocalizeKey("error.ui_initialization_failed", "Failed to initialize UI: %s"), err.Error()).Title(lang.LocalizeKey("app.error", "Error")).Error()
		return err
	}
	return nil
}

func startIPCListener(s *uicommon.State) {
	config := &winio.PipeConfig{
		MessageMode:      true,
		InputBufferSize:  4096,
		OutputBufferSize: 4096,
	}
	ln, err := winio.ListenPipe(pipeName, config)
	if err != nil {
		slog.Error("Failed to listen on pipe", "error", err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Error("Failed to accept pipe connection", "error", err)
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			reader := bufio.NewReader(c)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err != io.EOF {
						slog.Error("Failed to read from pipe", "error", err)
					}
					break
				}
				uri := strings.TrimSpace(line)
				if uri != "" && s.OnSharedURIReceived != nil {
					slog.Info("Received shared URI via IPC", "uri", uri)
					s.OnSharedURIReceived(uri)
				}
			}
		}(conn)
	}
}

func registerScheme() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\mod-of-us`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer key.Close()

	if err := key.SetStringValue("", "URL:Mod of Us Protocol"); err != nil {
		return err
	}
	if err := key.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}

	iconKey, _, err := registry.CreateKey(key, "DefaultIcon", registry.ALL_ACCESS)
	if err == nil {
		_ = iconKey.SetStringValue("", "\""+execPath+"\",0")
		iconKey.Close()
	}

	shellKey, _, err := registry.CreateKey(key, `shell\open\command`, registry.ALL_ACCESS)
	if err == nil {
		_ = shellKey.SetStringValue("", "\""+execPath+"\" \"%1\"")
		shellKey.Close()
	}

	return nil
}
