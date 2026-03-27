//go:build windows

package aumgr

import (
	"errors"
	"log/slog"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const TH32CS_SNAPPROCESS = 0x00000002
const processExitCodeStillActive = 259

type WindowsProcess struct {
	ProcessID       int
	ParentProcessID int
	Exe             string
}

func getProcesses() ([]WindowsProcess, error) {
	handle, err := windows.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := windows.CloseHandle(handle); err != nil {
			slog.Error("Failed to close process snapshot handle", "error", err)
		}
	}()

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	// get the first process
	err = windows.Process32First(handle, &entry)
	if err != nil {
		return nil, err
	}

	results := make([]WindowsProcess, 0, 50)
	for {
		results = append(results, newWindowsProcess(&entry))

		err = windows.Process32Next(handle, &entry)
		if err != nil {
			// windows sends ERROR_NO_MORE_FILES on last process
			if err == syscall.ERROR_NO_MORE_FILES {
				return results, nil
			}
			return nil, err
		}
	}
}

func findProcessByName(processes []WindowsProcess, name string) *WindowsProcess {
	for _, p := range processes {
		if strings.EqualFold(p.Exe, name) {
			return &p
		}
	}
	return nil
}

func newWindowsProcess(e *windows.ProcessEntry32) WindowsProcess {
	// Find when the string ends for decoding
	end := 0
	for e.ExeFile[end] != 0 {
		end++
	}

	return WindowsProcess{
		ProcessID:       int(e.ProcessID),
		ParentProcessID: int(e.ParentProcessID),
		Exe:             syscall.UTF16ToString(e.ExeFile[:end]),
	}
}

func IsAmongUsRunning() (pid int, err error) {
	processes, err := getProcesses()
	if err != nil {
		return 0, err
	}
	proc := findProcessByName(processes, "Among Us.exe")
	if proc == nil {
		return 0, nil
	}
	return proc.ProcessID, nil
}

func IsProcessRunning(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return false, nil
		}
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return true, nil
		}
		return false, err
	}
	defer func() {
		if err := windows.CloseHandle(handle); err != nil {
			slog.Warn("Failed to close process handle", "pid", pid, "error", err)
		}
	}()
	var code uint32
	if err := windows.GetExitCodeProcess(handle, &code); err != nil {
		return false, err
	}
	return code == processExitCodeStillActive, nil
}
