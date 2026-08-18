//go:build windows

package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"
	"github.com/Microsoft/go-winio"
	sdk "github.com/ikafly144/discord_social_sdk"
	"github.com/nightlyone/lockfile"
	"github.com/sqweek/dialog"
	"github.com/zzl/go-win32api/v2/win32"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/ikafly144/au_mod_installer/client/discord"
	"github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/client/ui"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
)

const AppUserModelID = "com.github.ikafly.au_mod_installer"

var DefaultServer = "https://modofus.sabafly.net/api/v1"
var pipeName = `\\.\pipe\au_mod_installer_ipc`

func main() {
	if hr := win32.SetCurrentProcessExplicitAppUserModelID(win32.StrToPwstr(AppUserModelID)); win32.FAILED(hr) {
		slog.Warn("SetCurrentProcessExplicitAppUserModelID returned error", "hresult", hr)
	}
	sharedURI := ""
	sharedArchive := ""
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "mod-of-us://") {
			sharedURI = arg
			break
		}
		if strings.EqualFold(filepath.Ext(arg), ".aupack") {
			sharedArchive = arg
			break
		}
	}

	pd, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if err != nil {
		slog.Error("Failed to get ProgramData folder path", "error", err)
		os.Exit(1)
	}
	lockPath := filepath.Join(pd, "au_mod_installer.lock")
	lock, err := lockfile.New(lockPath)
	if err != nil {
		slog.Error("Failed to create lockfile", "error", err)
		os.Exit(1)
	}
	err = lock.TryLock()
	if err != nil {
		slog.Error("Another instance is already running", "error", err)

		// Try to send URI to the existing instance via IPC
		conn, err := winio.DialPipe(pipeName, nil)
		if err == nil {
			defer conn.Close()
			if sharedURI != "" || sharedArchive != "" {
				if sharedURI != "" {
					_, _ = conn.Write([]byte("uri:" + sharedURI + "\n"))
					slog.Info("Sent shared URI to existing instance", "uri", sharedURI)
				}
				if sharedArchive != "" {
					absArchive, absErr := filepath.Abs(sharedArchive)
					if absErr == nil {
						_, _ = conn.Write([]byte("archive:" + absArchive + "\n"))
						slog.Info("Sent shared archive to existing instance", "path", absArchive)
					}
				}
			} else {
				_, _ = conn.Write([]byte("activate\n"))
				slog.Info("Sent activate command to existing instance")
			}

			_ = lock.Unlock()
			os.Exit(1)
		}

		// If pipe is not available, it means the existing instance is not available. So we can continue to run the new instance.
		slog.Warn("Failed to connect to existing instance via IPC", "error", err)
		if err := os.RemoveAll(lockPath); err != nil {
			slog.Error("Failed to remove lockfile", "error", err)
		}
		if err := lock.TryLock(); err != nil {
			slog.Error("Failed to acquire lockfile after removing it", "error", err)
			os.Exit(1)
		}
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			slog.Error("Failed to unlock lockfile", "error", err)
		}
	}()
	mainErr := realMain(sharedURI, sharedArchive)
	if mainErr != nil {
		os.Exit(1)
	}
}

func realMain(sharedURI string, sharedArchive string) error {
	var (
		localMode string
		server    string
		offline   bool
		silent    bool
		initial   bool
	)

	a := app.New()

	social := sdk.NewClient()
	activityService := discord.NewDiscordService(social)
	social.SetActivityJoinCallback(func(s string) {
		slog.Info("Received join activity callback", "uri", s)
		activityService.PushQueue(s)
	})
	social.AddLogCallback(func(arg0 string, arg1 sdk.LoggingSeverity) {
		level := slog.LevelInfo
		switch arg1 {
		case sdk.LoggingSeverityVerbose:
			level = slog.LevelDebug
		case sdk.LoggingSeverityWarning:
			level = slog.LevelWarn
		case sdk.LoggingSeverityError:
			level = slog.LevelError
		}
		slog.Default().With(slog.String("component", "discord_sdk")).Log(context.Background(), level, arg0)
	}, sdk.LoggingSeverityInfo)
	social.SetApplicationId(APPLICATION_ID)

	go func() {
		for {
			sdk.RunCallbacks()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	flag.StringVar(&localMode, "local", "", "Path to local mods.json file for local mode")
	flag.StringVar(&server, "server", DefaultServer, "URL of the mod server")
	flag.BoolVar(&offline, "offline", false, "Run in offline mode (only uninstallation and management of installed mods are available)")
	flag.BoolVar(&silent, "silent", false, "Start minimized in system tray")
	flag.BoolVar(&initial, "initial", false, "Indicates the application was launched from updater on startup")
	flag.Parse()

	if err := registerScheme(); err != nil {
		slog.Error("Failed to register scheme", "error", err)
	}

	uicommon.SyncAutoStart(
		a.Preferences().BoolWithFallback("launch_on_startup", true),
		a.Preferences().BoolWithFallback("start_silent", true),
	)

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

	w := a.NewWindow(lang.LocalizeKey("app.name", "Mod of Us") + " " + version)
	if path, err := os.Executable(); err == nil {

		social.RegisterLaunchCommand(APPLICATION_ID, path)
	}

	activityService.SetIdleActivity(func() *sdk.Activity {
		act := sdk.NewActivity()
		act.SetType(sdk.ActivityTypesPlaying)
		act.SetName("Mod of Us")
		act.SetState(lang.LocalizeKey("discord.status.idle", "Idle"))
		act.SetDetails(lang.LocalizeKey("discord.status.idle_details", "Not currently running the game"))
		assets := sdk.NewActivityAssets()
		assets.SetLargeImage("icon")
		if version != "" {
			assets.SetLargeText(fmt.Sprintf("Mod of Us %s", version))
		} else {
			assets.SetLargeText("Mod of Us")
		}
		act.SetAssets(assets)
		return act
	}, func(d *sdk.ClientResult) {
		if !d.Successful() {
			slog.Warn("Failed to set idle activity", "error", d.ErrorCode())
		}
	})

	if err := ui.Main(w, version, sharedURI, sharedArchive,
		ui.WithSilent(silent),
		ui.WithStateOptions(
			uicommon.WithRestClient(client),
			uicommon.WithActivityService(activityService),
			uicommon.WithInitial(initial),
		),
		ui.WithStateInit(func(s *uicommon.State) {
			go startIPCListener(s)
		}),
	); err != nil {
		slog.Error("Failed to initialize UI", "error", err)
		dialog.Message(lang.LocalizeKey("error.ui_initialization_failed", "Failed to initialize UI: %s"), err.Error()).Title(lang.LocalizeKey("app.error", "Error")).Error()
		return err
	}

	runtime.KeepAlive(social)
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

	var closeOnce sync.Once
	closePipe := func() {
		closeOnce.Do(func() {
			slog.Info("Closing IPC pipe listener")
			_ = ln.Close()
		})
	}
	s.CloseIPC = closePipe
	defer closePipe()

	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Info("IPC listener accept loop ended", "error", err)
			break
		}
		go func(c net.Conn) {
			defer c.Close()
			reader := bufio.NewReader(c)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err != io.EOF && !errors.Is(err, net.ErrClosed) {
						slog.Error("Failed to read from pipe", "error", err)
					}
					break
				}
				message := strings.TrimSpace(line)
				if message == "" {
					continue
				}
				switch {
				case strings.HasPrefix(message, "uri:"):
					uri := strings.TrimPrefix(message, "uri:")
					if uri != "" && s.OnSharedURIReceived != nil {
						slog.Info("Received shared URI via IPC", "uri", uri)
						s.OnSharedURIReceived(uri)
					}
				case strings.HasPrefix(message, "archive:"):
					path := strings.TrimPrefix(message, "archive:")
					if path != "" && s.OnSharedArchiveReceived != nil {
						slog.Info("Received shared archive path via IPC", "path", path)
						s.OnSharedArchiveReceived(path)
					}
				case message == "activate":
					slog.Info("Received activate command via IPC")
					if s.OnActivateReceived != nil {
						s.OnActivateReceived()
					}
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

	extKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\.aupack`, registry.ALL_ACCESS)
	if err == nil {
		_ = extKey.SetStringValue("", "mod-of-us.aupack")
		extKey.Close()
	}

	fileTypeKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\mod-of-us.aupack`, registry.ALL_ACCESS)
	if err == nil {
		_ = fileTypeKey.SetStringValue("", "Mod of Us Archive")
		iconKey, _, iconErr := registry.CreateKey(fileTypeKey, "DefaultIcon", registry.ALL_ACCESS)
		if iconErr == nil {
			_ = iconKey.SetStringValue("", "\""+execPath+"\",0")
			iconKey.Close()
		}
		openCommandKey, _, openErr := registry.CreateKey(fileTypeKey, `shell\open\command`, registry.ALL_ACCESS)
		if openErr == nil {
			_ = openCommandKey.SetStringValue("", "\""+execPath+"\" \"%1\"")
			openCommandKey.Close()
		}
		fileTypeKey.Close()
	}

	return nil
}
