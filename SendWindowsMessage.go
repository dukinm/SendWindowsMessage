package SendWindowsMessage

import (
	"errors"
	"github.com/gonutz/ide/w32"
	"github.com/hallazzang/go-windows-programming/pkg/win"
	"golang.org/x/sys/windows"
	"log"
	"math/rand"
	"time"
	"unsafe"
)

// <a href="https://iconscout.com/icons/qr-code" target="_blank">Qr Code Icon</a> by <a href="https://iconscout.com/contributors/iconscout" target="_blank">Iconscout Store</a>
// String returns a human-friendly display name of the hotkey
// such as "Hotkey[Id: 1, Alt+Ctrl+O]"
var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procSetWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessage          = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessage     = user32.NewProc("DispatchMessageW")
	keyboardHook            HHOOK
)

const notifyIconMsg = win.WM_APP + 1

var errShellNotifyIcon = errors.New("Shell_NotifyIcon error")

func init() {
	rand.Seed(time.Now().UnixNano())
}

type NotifyIcon struct {
	hwnd uintptr
	guid win.GUID
}

func hideConsole() {
	console := w32.GetConsoleWindow()
	if console == 0 {
		return // no console attached
	}
	// If this application is the process that created the console window, then
	// this program was not compiled with the -H=windowsgui flag and on start-up
	// it created a console along with the main application window. In this case
	// hide the console window.
	// See
	// http://stackoverflow.com/questions/9009333/how-to-check-if-the-program-is-run-from-a-console
	_, consoleProcID := w32.GetWindowThreadProcessId(console)
	if w32.GetCurrentProcessId() == consoleProcID {
		w32.ShowWindowAsync(console, w32.SW_HIDE)
	}
}
func newNotifyIcon(hwnd uintptr) (*NotifyIcon, error) {
	ni := &NotifyIcon{
		hwnd: hwnd,
		guid: newGUID(),
	}
	data := ni.newData()
	data.UFlags |= win.NIF_MESSAGE
	data.UCallbackMessage = notifyIconMsg
	if win.Shell_NotifyIcon(win.NIM_ADD, data) == win.FALSE {
		return nil, errShellNotifyIcon
	}
	return ni, nil
}

func (ni *NotifyIcon) Dispose() {
	win.Shell_NotifyIcon(win.NIM_DELETE, ni.newData())
}

func (ni *NotifyIcon) SetTooltip(tooltip string) error {
	data := ni.newData()
	data.UFlags |= win.NIF_TIP
	copy(data.SzTip[:], windows.StringToUTF16(tooltip))
	if win.Shell_NotifyIcon(win.NIM_MODIFY, data) == win.FALSE {
		return errShellNotifyIcon
	}
	return nil
}

func (ni *NotifyIcon) SetIcon(hIcon uintptr) error {
	data := ni.newData()
	data.UFlags |= win.NIF_ICON
	data.HIcon = hIcon
	if win.Shell_NotifyIcon(win.NIM_MODIFY, data) == win.FALSE {
		return errShellNotifyIcon
	}
	return nil
}

func (ni *NotifyIcon) ShowNotification(title, text string) error {
	data := ni.newData()
	data.UFlags |= win.NIF_INFO
	copy(data.SzInfoTitle[:], windows.StringToUTF16(title))
	copy(data.SzInfo[:], windows.StringToUTF16(text))
	if win.Shell_NotifyIcon(win.NIM_MODIFY, data) == win.FALSE {
		return errShellNotifyIcon
	}
	return nil
}

func (ni *NotifyIcon) ShowNotificationWithIcon(title, text string, hIcon uintptr) error {
	data := ni.newData()
	data.UFlags |= win.NIF_INFO
	copy(data.SzInfoTitle[:], windows.StringToUTF16(title))
	copy(data.SzInfo[:], windows.StringToUTF16(text))
	data.DwInfoFlags = win.NIIF_USER | win.NIIF_LARGE_ICON
	if win.Shell_NotifyIcon(win.NIM_MODIFY, data) == win.FALSE {
		return errShellNotifyIcon
	}
	return nil
}

func (ni *NotifyIcon) newData() *win.NOTIFYICONDATA {
	var nid win.NOTIFYICONDATA
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.UFlags = win.NIF_GUID
	nid.HWnd = ni.hwnd
	nid.GuidItem = ni.guid
	return &nid
}

func newGUID() win.GUID {
	var buf [16]byte
	rand.Read(buf[:])
	return *(*win.GUID)(unsafe.Pointer(&buf[0]))
}

type (
	DWORD     uint32
	WPARAM    uintptr
	LPARAM    uintptr
	LRESULT   uintptr
	HANDLE    uintptr
	HINSTANCE HANDLE
	HHOOK     HANDLE
	HWND      HANDLE
)

type HOOKPROC func(int, WPARAM, LPARAM) LRESULT

type KBDLLHOOKSTRUCT struct {
	VkCode      DWORD
	ScanCode    DWORD
	Flags       DWORD
	Time        DWORD
	DwExtraInfo uintptr
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd162805.aspx
type POINT struct {
	X, Y int32
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/ms644958.aspx
type MSG struct {
	Hwnd    HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

func wndProc(hWnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case notifyIconMsg:
		switch nmsg := win.LOWORD(uint32(lParam)); nmsg {
		case win.NIN_BALLOONUSERCLICK:
			log.Print("User has clicked the balloon message")
		case win.WM_LBUTTONDOWN:
			//clickHandler(hWnd,msg)
			win.PostQuitMessage(0)

		}
	case win.WM_DESTROY:
		win.PostQuitMessage(0)
	default:
		return win.DefWindowProc(hWnd, msg, wParam, lParam)
	}
	return 0
}

func CreateMainWindow() (uintptr, error) {
	hInstance := win.GetModuleHandle(nil)

	wndClass := windows.StringToUTF16Ptr("MyWindow")

	var wcex win.WNDCLASSEX
	wcex.CbSize = uint32(unsafe.Sizeof(wcex))
	wcex.Style = win.CS_HREDRAW | win.CS_VREDRAW
	wcex.LpfnWndProc = windows.NewCallback(wndProc)
	wcex.HInstance = hInstance
	wcex.HCursor = win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_ARROW))
	wcex.HbrBackground = win.COLOR_WINDOW + 1
	wcex.LpszClassName = wndClass
	if win.RegisterClassEx(&wcex) == 0 {
		return 0, win.GetLastError()
	}

	hwnd := win.CreateWindowEx(0, wndClass, windows.StringToUTF16Ptr("NotifyIcon Example"), win.WS_THICKFRAME, -100, -100, 0, 0, 0, 0, hInstance, nil)
	if hwnd == win.NULL {
		return 0, win.GetLastError()
	}
	win.ShowWindow(hwnd, win.SW_SHOW)

	return hwnd, nil
}

func loadIconFromResource(id uintptr) (uintptr, error) {
	hIcon := win.LoadImage(
		win.GetModuleHandle(nil),
		win.MAKEINTRESOURCE(id),
		win.IMAGE_ICON,
		0, 0,
		win.LR_DEFAULTSIZE)
	if hIcon == win.NULL {
		return 0, win.GetLastError()
	}

	return hIcon, nil
}
func loadIconFromFile(name string) (uintptr, error) {
	hIcon := win.LoadImage(
		win.NULL,
		windows.StringToUTF16Ptr(name),
		win.IMAGE_ICON,
		0, 0,
		win.LR_DEFAULTSIZE|win.LR_LOADFROMFILE)
	if hIcon == win.NULL {
		return 0, win.GetLastError()
	}

	return hIcon, nil
}

//func clickHandler(hWnd uintptr, msg string) {
//	fmt.Println("User has clicked the notify icon")
//	fmt.Println(msg)
//	win.DestroyIcon(hWnd);
//}

func start(message string, tooltip string, icon string, ni *NotifyIcon) {

	//var msg win.MSG
	// defer user32.Release()
	needSendNotificationWithIcon := true

	if len(icon) == 0 {
		needSendNotificationWithIcon = false
	}
	var hIcon uintptr
	var err error
	if needSendNotificationWithIcon {

		hIcon, err = loadIconFromResource(10) // rsrc uses 10 for icon resource id
		if err != nil {
			hIcon, err = loadIconFromFile(icon) // fallback to use file
			if err != nil {
				needSendNotificationWithIcon = false
			} else {
				defer win.DestroyIcon(hIcon)
			}
		}
	}

	//hwnd, err := createMainWindow()
	//if err != nil {
	//	panic(err)
	//}

	ni.SetTooltip(tooltip)
	if needSendNotificationWithIcon {
		ni.SetIcon(hIcon)
		ni.ShowNotificationWithIcon(tooltip, message, hIcon)
	} else {
		ni.ShowNotification(tooltip, message)
	}
	//defer ni.Dispose()
	//ni.ShowNotificationWithIcon(tooltip, message,nil)
	//for win.GetMessage(&msg, 0, 0, 0) != 0 {
	//	win.TranslateMessage(&msg)
	//	win.DispatchMessage(&msg)
	//}

}

func CreateSender() *NotifyIcon {
	windowId, err := CreateMainWindow()
	if err != nil {
		panic(err)
	}
	ni, err := newNotifyIcon(windowId)
	if err != nil {
		panic(err)
	}
	return ni

}

func SendMessage(message string, tooltip string, icon string, ni *NotifyIcon) {

	hideConsole()
	start(message, tooltip, icon, ni)

}
