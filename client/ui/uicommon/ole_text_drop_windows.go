//go:build windows

package uicommon

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/zzl/go-win32api/v2/win32"
)

type oleTextDropTarget struct {
	lpVtbl     *oleTextDropTargetVtbl
	refCount   int32
	onDropText func(string)
}

type oleTextDropTargetVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	DragEnter      uintptr
	DragOver       uintptr
	DragLeave      uintptr
	Drop           uintptr
}

var globalOleTextDropTargetVtbl = oleTextDropTargetVtbl{
	QueryInterface: syscall.NewCallback(oleDropTargetQueryInterface),
	AddRef:         syscall.NewCallback(oleDropTargetAddRef),
	Release:        syscall.NewCallback(oleDropTargetRelease),
	DragEnter:      syscall.NewCallback(oleDropTargetDragEnter),
	DragOver:       syscall.NewCallback(oleDropTargetDragOver),
	DragLeave:      syscall.NewCallback(oleDropTargetDragLeave),
	Drop:           syscall.NewCallback(oleDropTargetDrop),
}

func (s *State) EnableNativeTextDrop() (func(), error) {
	if s.Window == nil {
		return nil, fmt.Errorf("window is nil")
	}

	hwnd, err := nativeHWNDFromFyneWindow(s.Window)
	if err != nil {
		return nil, fmt.Errorf("failed to get HWND for native text drop: %w", err)
	}

	hr := win32.OleInitialize(nil)
	if hr < 0 {
		return nil, fmt.Errorf("OleInitialize failed: 0x%08X", uint32(hr))
	}

	target := &oleTextDropTarget{
		lpVtbl:   &globalOleTextDropTargetVtbl,
		refCount: 1,
		onDropText: func(text string) {
			s.handleOLEDroppedText(text)
		},
	}
	hr = win32.RegisterDragDrop(win32.HWND(hwnd), (*win32.IDropTarget)(unsafe.Pointer(target)))
	if hr < 0 {
		win32.OleUninitialize()
		return nil, fmt.Errorf("RegisterDragDrop failed: 0x%08X", uint32(hr))
	}

	cleanup := func() {
		_ = target // keep target alive while registered
		_ = win32.RevokeDragDrop(win32.HWND(hwnd))
		win32.OleUninitialize()
	}
	return cleanup, nil
}

func (s *State) handleOLEDroppedText(text string) {
	slog.Info("Received OLE dropped text", "text", text)
	for line := range strings.SplitSeq(strings.ReplaceAll(text, "\r", "\n"), "\n") {
		token := strings.TrimSpace(line)
		if token == "" {
			continue
		}
		if archivePath, ok := parseDroppedArchivePath(token); ok {
			if s.OnSharedArchiveReceived != nil {
				s.OnSharedArchiveReceived(archivePath)
			}
			return
		}
		if parsedURI, ok := parseDroppedURI(token); ok {
			if s.OnSharedURIReceived != nil {
				s.OnSharedURIReceived(parsedURI)
			}
			return
		}
	}
}

func parseDroppedArchivePath(token string) (string, bool) {
	token = normalizeDroppedToken(token)
	if token == "" {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(token), "http://") || strings.HasPrefix(strings.ToLower(token), "https://") {
		return "", false
	}

	if strings.HasPrefix(strings.ToLower(token), "file://") {
		u, err := url.Parse(token)
		if err != nil {
			return "", false
		}
		decodedPath, err := url.PathUnescape(u.Path)
		if err != nil {
			decodedPath = u.Path
		}

		var candidate string
		switch {
		case u.Host != "" && !strings.EqualFold(u.Host, "localhost"):
			p := filepath.FromSlash(decodedPath)
			p = strings.TrimPrefix(p, `\`)
			candidate = `\\` + u.Host + `\` + p
		default:
			p := filepath.FromSlash(decodedPath)
			if len(p) >= 3 && p[0] == '\\' && p[2] == ':' {
				p = p[1:]
			}
			candidate = p
		}
		if strings.EqualFold(filepath.Ext(candidate), ".aupack") {
			if absPath, err := filepath.Abs(candidate); err == nil {
				return absPath, true
			}
			return candidate, true
		}
		return "", false
	}

	if !strings.EqualFold(filepath.Ext(token), ".aupack") {
		return "", false
	}
	if absPath, err := filepath.Abs(token); err == nil {
		return absPath, true
	}
	return token, true
}

func parseDroppedURI(token string) (string, bool) {
	token = normalizeDroppedToken(token)
	if token == "" {
		return "", false
	}
	u, err := url.Parse(token)
	if err != nil {
		return "", false
	}
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		return "", false
	}
	if u.Host == "" {
		return "", false
	}
	return u.String(), true
}

func normalizeDroppedToken(token string) string {
	token = strings.TrimSpace(strings.Trim(token, "\""))
	if len(token) >= 4 && strings.EqualFold(token[:4], "URL=") {
		token = strings.TrimSpace(token[4:])
	}
	return token
}

func dataObjectHasText(dataObj *win32.IDataObject) bool {
	if dataObj == nil {
		return false
	}
	if queryDataObjectByFormat(dataObj, uint16(win32.CF_UNICODETEXT)) {
		return true
	}
	if cf := registerClipboardFormatID("UniformResourceLocatorW"); cf != 0 && queryDataObjectByFormat(dataObj, cf) {
		return true
	}
	if cf := registerClipboardFormatID("UniformResourceLocator"); cf != 0 && queryDataObjectByFormat(dataObj, cf) {
		return true
	}
	return false
}

func extractTextFromDataObject(dataObj *win32.IDataObject) (string, bool) {
	if dataObj == nil {
		return "", false
	}
	if text, ok := extractUTF16TextFromDataObject(dataObj, uint16(win32.CF_UNICODETEXT)); ok {
		return text, true
	}
	if cf := registerClipboardFormatID("UniformResourceLocatorW"); cf != 0 {
		if text, ok := extractUTF16TextFromDataObject(dataObj, cf); ok {
			return text, true
		}
	}
	if cf := registerClipboardFormatID("UniformResourceLocator"); cf != 0 {
		if text, ok := extractANSITextFromDataObject(dataObj, cf); ok {
			return text, true
		}
	}
	return "", false
}

func utf16PtrToString(ptr *uint16) string {
	if ptr == nil {
		return ""
	}
	chars := make([]uint16, 0, 128)
	for offset := uintptr(0); ; offset += 2 {
		ch := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + offset))
		if ch == 0 {
			break
		}
		chars = append(chars, ch)
	}
	return syscall.UTF16ToString(chars)
}

//nolint:govet // COM callback pointers are raw Win32 pointers passed as uintptr.
func oleDropTargetQueryInterface(this, riidPtr, ppvPtr uintptr) uintptr {
	defer recoverOleDropCallback()
	if ppvPtr == 0 {
		return hresultToUintptr(win32.E_FAIL)
	}
	ppv := (*unsafe.Pointer)(unsafe.Pointer(ppvPtr))
	*ppv = nil
	if riidPtr == 0 {
		return hresultToUintptr(win32.E_NOINTERFACE)
	}
	riid := (*syscall.GUID)(unsafe.Pointer(riidPtr))
	if *riid == win32.IID_IUnknown || *riid == win32.IID_IDropTarget {
		*ppv = unsafe.Pointer(this)
		oleDropTargetAddRef(this)
		return hresultToUintptr(win32.S_OK)
	}
	return hresultToUintptr(win32.E_NOINTERFACE)
}

//nolint:govet // COM callback receives object pointer as uintptr from Win32.
func oleDropTargetAddRef(this uintptr) uintptr {
	defer recoverOleDropCallback()
	target := (*oleTextDropTarget)(unsafe.Pointer(this))
	return uintptr(atomic.AddInt32(&target.refCount, 1))
}

//nolint:govet // COM callback receives object pointer as uintptr from Win32.
func oleDropTargetRelease(this uintptr) uintptr {
	defer recoverOleDropCallback()
	target := (*oleTextDropTarget)(unsafe.Pointer(this))
	return uintptr(atomic.AddInt32(&target.refCount, -1))
}

//nolint:govet // COM callback receives effect/data pointers as uintptr from Win32.
func oleDropTargetDragEnter(this, dataObjPtr, _ uintptr, _ uintptr, effectPtr uintptr) uintptr {
	defer recoverOleDropCallback()
	if effectPtr == 0 {
		return hresultToUintptr(win32.S_OK)
	}
	effect := (*uint32)(unsafe.Pointer(effectPtr))
	if dataObjectHasText((*win32.IDataObject)(unsafe.Pointer(dataObjPtr))) {
		*effect = uint32(win32.DROPEFFECT_COPY)
		return hresultToUintptr(win32.S_OK)
	}
	*effect = uint32(win32.DROPEFFECT_NONE)
	return hresultToUintptr(win32.S_OK)
}

//nolint:govet // COM callback receives effect pointer as uintptr from Win32.
func oleDropTargetDragOver(_, _, _, effectPtr uintptr) uintptr {
	defer recoverOleDropCallback()
	if effectPtr == 0 {
		return hresultToUintptr(win32.S_OK)
	}
	effect := (*uint32)(unsafe.Pointer(effectPtr))
	*effect = uint32(win32.DROPEFFECT_COPY)
	return hresultToUintptr(win32.S_OK)
}

func oleDropTargetDragLeave(uintptr) uintptr {
	defer recoverOleDropCallback()
	return hresultToUintptr(win32.S_OK)
}

//nolint:govet // COM callback receives object/effect/data pointers as uintptr from Win32.
func oleDropTargetDrop(this, dataObjPtr, _ uintptr, _ uintptr, effectPtr uintptr) uintptr {
	defer recoverOleDropCallback()
	if effectPtr == 0 {
		return hresultToUintptr(win32.S_OK)
	}
	effect := (*uint32)(unsafe.Pointer(effectPtr))
	target := (*oleTextDropTarget)(unsafe.Pointer(this))
	text, ok := extractTextFromDataObject((*win32.IDataObject)(unsafe.Pointer(dataObjPtr)))
	if !ok {
		*effect = uint32(win32.DROPEFFECT_NONE)
		return hresultToUintptr(win32.S_OK)
	}
	if target.onDropText != nil {
		target.onDropText(text)
	}
	*effect = uint32(win32.DROPEFFECT_COPY)
	return hresultToUintptr(win32.S_OK)
}

func hresultToUintptr(hr win32.HRESULT) uintptr {
	return uintptr(uint32(hr))
}

func queryDataObjectByFormat(dataObj *win32.IDataObject, format uint16) bool {
	formatEtc := win32.FORMATETC{
		CfFormat: format,
		DwAspect: uint32(win32.DVASPECT_CONTENT),
		Lindex:   -1,
		Tymed:    uint32(win32.TYMED_HGLOBAL),
	}
	return dataObj.QueryGetData(&formatEtc) >= 0
}

func extractUTF16TextFromDataObject(dataObj *win32.IDataObject, format uint16) (string, bool) {
	formatEtc := win32.FORMATETC{
		CfFormat: format,
		DwAspect: uint32(win32.DVASPECT_CONTENT),
		Lindex:   -1,
		Tymed:    uint32(win32.TYMED_HGLOBAL),
	}
	var medium win32.STGMEDIUM
	if dataObj.GetData(&formatEtc, &medium) < 0 {
		return "", false
	}
	defer win32.ReleaseStgMedium(&medium)

	if medium.Tymed != win32.TYMED_HGLOBAL {
		return "", false
	}
	hGlobal := medium.HGlobalVal()
	if hGlobal == 0 {
		return "", false
	}
	ptr, _ := win32.GlobalLock(hGlobal)
	if ptr == nil {
		return "", false
	}
	defer func() {
		_, _ = win32.GlobalUnlock(hGlobal)
	}()

	text := utf16PtrToString((*uint16)(ptr))
	return text, strings.TrimSpace(text) != ""
}

func extractANSITextFromDataObject(dataObj *win32.IDataObject, format uint16) (string, bool) {
	formatEtc := win32.FORMATETC{
		CfFormat: format,
		DwAspect: uint32(win32.DVASPECT_CONTENT),
		Lindex:   -1,
		Tymed:    uint32(win32.TYMED_HGLOBAL),
	}
	var medium win32.STGMEDIUM
	if dataObj.GetData(&formatEtc, &medium) < 0 {
		return "", false
	}
	defer win32.ReleaseStgMedium(&medium)

	if medium.Tymed != win32.TYMED_HGLOBAL {
		return "", false
	}
	hGlobal := medium.HGlobalVal()
	if hGlobal == 0 {
		return "", false
	}
	ptr, _ := win32.GlobalLock(hGlobal)
	if ptr == nil {
		return "", false
	}
	defer func() {
		_, _ = win32.GlobalUnlock(hGlobal)
	}()

	size, _ := win32.GlobalSize(hGlobal)
	if size == 0 {
		return "", false
	}
	raw := unsafe.Slice((*byte)(ptr), size)
	if len(raw) == 0 {
		return "", false
	}
	if i := bytes.IndexByte(raw, 0); i >= 0 {
		raw = raw[:i]
	}
	text := strings.TrimSpace(string(raw))
	return text, text != ""
}

func registerClipboardFormatID(name string) uint16 {
	utf16Name, err := syscall.UTF16PtrFromString(name)
	if err != nil || utf16Name == nil {
		return 0
	}
	format, regErr := win32.RegisterClipboardFormat(utf16Name)
	if regErr != win32.WIN32_ERROR(0) || format == 0 {
		return 0
	}
	return uint16(format)
}

func recoverOleDropCallback() {
	if r := recover(); r != nil {
		stack := debug.Stack()
		_ = stack
	}
}
