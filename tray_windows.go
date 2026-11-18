//go:build windows

package main

// Native Win32 notification-area integration, without an installer or CGO.
// The browser remains the TV window; closing it never shuts down the server.
import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

const (
	wmDestroy       = 0x0002
	wmCommand       = 0x0111
	wmClose         = 0x0010
	wmNull          = 0x0000
	wmRButtonUp     = 0x0205
	wmLButtonDblClk = 0x0203
	wmContextMenu   = 0x007B
	wmApp           = 0x8000
	trayMessage     = wmApp + 21
	nimAdd          = 0
	nimModify       = 1
	nimDelete       = 2
	nimSetVersion   = 4
	nifMessage      = 1
	nifIcon         = 2
	nifTip          = 4
	notifyVersion4  = 4
	mfString        = 0
	mfSeparator     = 0x0800
	tpmRightButton  = 0x0002
	tpmReturnCmd    = 0x0100
	imageIcon       = 1
	lrDefaultColor  = 0
	menuOpen        = 101
	menuSettings    = 102
	menuStatus      = 103
	menuLogs        = 104
	menuExit        = 199
)

var (
	userDLL               = syscall.NewLazyDLL("user32.dll")
	shellDLL              = syscall.NewLazyDLL("shell32.dll")
	kernelDLL             = syscall.NewLazyDLL("kernel32.dll")
	registerClassEx       = userDLL.NewProc("RegisterClassExW")
	createWindowEx        = userDLL.NewProc("CreateWindowExW")
	defWindowProc         = userDLL.NewProc("DefWindowProcW")
	destroyWindow         = userDLL.NewProc("DestroyWindow")
	getMessage            = userDLL.NewProc("GetMessageW")
	translateMessage      = userDLL.NewProc("TranslateMessage")
	dispatchMessage       = userDLL.NewProc("DispatchMessageW")
	postQuitMessage       = userDLL.NewProc("PostQuitMessage")
	postMessage           = userDLL.NewProc("PostMessageW")
	registerWindowMessage = userDLL.NewProc("RegisterWindowMessageW")
	loadImage             = userDLL.NewProc("LoadImageW")
	loadIcon              = userDLL.NewProc("LoadIconW")
	createPopupMenu       = userDLL.NewProc("CreatePopupMenu")
	appendMenu            = userDLL.NewProc("AppendMenuW")
	trackPopupMenu        = userDLL.NewProc("TrackPopupMenu")
	destroyMenu           = userDLL.NewProc("DestroyMenu")
	setForegroundWindow   = userDLL.NewProc("SetForegroundWindow")
	getCursorPos          = userDLL.NewProc("GetCursorPos")
	showMessage           = userDLL.NewProc("MessageBoxW")
	shellNotifyIcon       = shellDLL.NewProc("Shell_NotifyIconW")
	moduleHandle          = kernelDLL.NewProc("GetModuleHandleW")
)

type winPoint struct{ x, y int32 }
type winMsg struct {
	hwnd           uintptr
	message        uint32
	_              uint32
	wParam, lParam uintptr
	time           uint32
	pt             winPoint
	private        uint32
}
type winClass struct {
	size, style                                                    uint32
	windowProc                                                     uintptr
	classExtra, windowExtra                                        int32
	instance, icon, cursor, background, menu, className, iconSmall uintptr
}
type notifyIcon struct {
	size                       uint32
	hwnd                       uintptr
	id, flags, callbackMessage uint32
	icon                       uintptr
	tip                        [128]uint16
	state, stateMask           uint32
	info                       [256]uint16
	version                    uint32
	infoTitle                  [64]uint16
	infoFlags                  uint32
	guid                       [16]byte
	balloonIcon                uintptr
}
type trayState struct {
	window                      uintptr
	tvURL, settingsURL, logFile string
	stop                        func()
	icon                        uintptr
	explorerMessage             uintptr
	closed                      bool
}

var activeTray *trayState

func utf16(s string) *uint16 { return syscall.StringToUTF16Ptr(s) }
func nativeAlert(title, message string) {
	showMessage.Call(0, uintptr(unsafe.Pointer(utf16(message))), uintptr(unsafe.Pointer(utf16(title))), 0x10)
}
func notify(state *trayState, cmd uintptr) bool {
	n := notifyIcon{size: uint32(unsafe.Sizeof(notifyIcon{})), hwnd: state.window, id: 1, flags: nifMessage | nifIcon | nifTip, callbackMessage: trayMessage, icon: state.icon}
	copy(n.tip[:], syscall.StringToUTF16("AussieTV - right-click for options"))
	r, _, _ := shellNotifyIcon.Call(cmd, uintptr(unsafe.Pointer(&n)))
	if r == 0 {
		return false
	}
	if cmd == nimAdd {
		n.version = notifyVersion4
		shellNotifyIcon.Call(nimSetVersion, uintptr(unsafe.Pointer(&n)))
	}
	return true
}
func appendItem(h uintptr, id uintptr, title string) {
	appendMenu.Call(h, mfString, id, uintptr(unsafe.Pointer(utf16(title))))
}
func handleTrayMenu(s *trayState) {
	menu, _, _ := createPopupMenu.Call()
	if menu == 0 {
		return
	}
	defer destroyMenu.Call(menu)
	appendItem(menu, menuOpen, "Open AussieTV")
	appendItem(menu, menuSettings, "Settings...")
	appendItem(menu, menuStatus, "Jellyfin feed status")
	appendItem(menu, menuLogs, "Open startup log")
	appendMenu.Call(menu, mfSeparator, 0, 0)
	appendItem(menu, menuExit, "Exit AussieTV")
	var p winPoint
	getCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	setForegroundWindow.Call(s.window)
	id, _, _ := trackPopupMenu.Call(menu, tpmRightButton|tpmReturnCmd, uintptr(p.x), uintptr(p.y), 0, s.window, 0)
	postMessage.Call(s.window, wmNull, 0, 0)
	switch id {
	case menuOpen:
		if err := openBrowser(s.tvURL); err != nil {
			log.Printf("tray open: %v", err)
		}
	case menuSettings, menuStatus:
		if err := openBrowser(s.settingsURL); err != nil {
			log.Printf("tray settings: %v", err)
		}
	case menuLogs:
		if err := exec.Command("notepad.exe", s.logFile).Start(); err != nil {
			log.Printf("tray logs: %v", err)
		}
	case menuExit:
		destroyWindow.Call(s.window)
	}
}
func trayWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	s := activeTray
	if s != nil {
		if msg == uint32(s.explorerMessage) && s.explorerMessage != 0 {
			notify(s, nimAdd)
			return 0
		}
		if msg == trayMessage {
			// NOTIFYICON_VERSION_4 places the event in LOWORD(lParam).
			action := uint32(lParam) & 0xFFFF
			switch action {
			case wmRButtonUp, wmContextMenu:
				handleTrayMenu(s)
			case wmLButtonDblClk:
				if err := openBrowser(s.tvURL); err != nil {
					log.Print(err)
				}
			}
			return 0
		}
		if msg == wmClose {
			destroyWindow.Call(hwnd)
			return 0
		}
		if msg == wmDestroy {
			if !s.closed {
				s.closed = true
				notify(s, nimDelete)
				go s.stop()
			}
			postQuitMessage.Call(0)
			return 0
		}
	}
	res, _, _ := defWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return res
}
func startTray(tvURL, settingsURL, logfile string, stop func()) error {
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		inst, _, _ := moduleHandle.Call(0)
		className := utf16("AussieTVTrayMessageWindow")
		callback := syscall.NewCallback(trayWndProc)
		wc := winClass{size: uint32(unsafe.Sizeof(winClass{})), windowProc: callback, instance: inst, className: uintptr(unsafe.Pointer(className))}
		if r, _, e := registerClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
			ready <- fmt.Errorf("register notification-area window: %v", e)
			return
		}
		title := utf16("AussieTV")
		hwnd, _, e := createWindowEx.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 0, 0, 0, 0, inst, 0)
		if hwnd == 0 {
			ready <- fmt.Errorf("create notification-area window: %v", e)
			return
		}
		icon, _, _ := loadImage.Call(inst, 1, imageIcon, 0, 0, lrDefaultColor)
		if icon == 0 {
			icon, _, _ = loadIcon.Call(0, 32512)
		}
		if icon == 0 {
			destroyWindow.Call(hwnd)
			ready <- fmt.Errorf("cannot load tray icon")
			return
		}
		file := logfile
		if file == "" {
			file = filepath.Join(os.Getenv("LOCALAPPDATA"), "AussieTV", "startup.log")
		}
		taskbar, _, _ := registerWindowMessage.Call(uintptr(unsafe.Pointer(utf16("TaskbarCreated"))))
		state := &trayState{window: hwnd, tvURL: tvURL, settingsURL: settingsURL, logFile: file, stop: stop, icon: icon, explorerMessage: taskbar}
		activeTray = state
		if !notify(state, nimAdd) {
			destroyWindow.Call(hwnd)
			ready <- fmt.Errorf("Windows notification-area icon could not be added")
			return
		}
		ready <- nil
		var msg winMsg
		for {
			r, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(r) <= 0 {
				break
			}
			translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
		activeTray = nil
	}()
	// A broken shell/notification service must not hold the entire player
	// before the local HTTP server even begins accepting requests.
	select {
	case err := <-ready:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("Windows tray initialization timed out; starting browser fallback")
	}
}
