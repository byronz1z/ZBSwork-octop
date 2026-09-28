//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	user32DLL = syscall.NewLazyDLL("user32.dll")

	procReleaseCapture  = user32DLL.NewProc("ReleaseCapture")
	procPostMessageW    = user32DLL.NewProc("PostMessageW")
	procGetWindowRect   = user32DLL.NewProc("GetWindowRect")
	procGetCursorPos    = user32DLL.NewProc("GetCursorPos")
	procGetDpiForWindow = user32DLL.NewProc("GetDpiForWindow")
)

const w32WMNCLButtonDown = 0x00A1 // WM_NCLBUTTONDOWN

// w32HitTestCodes mirrors wails' edgeMap in
// pkg/application/webview_window_windows.go so the WM_NCLBUTTONDOWN posted by
// startWindowResize matches what wails itself posts for wails:resize:<edge>.
var w32HitTestCodes = map[string]uintptr{
	"n-resize":  12, // HTTOP
	"ne-resize": 14, // HTTOPRIGHT
	"e-resize":  11, // HTRIGHT
	"se-resize": 17, // HTBOTTOMRIGHT
	"s-resize":  15, // HTBOTTOM
	"sw-resize": 16, // HTBOTTOMLEFT
	"w-resize":  10, // HTLEFT
	"nw-resize": 13, // HTTOPLEFT
}

type w32Rect struct {
	left, top, right, bottom int32
}

type w32Point struct {
	x, y int32
}

func user32ReleaseCapture() bool {
	ret, _, _ := procReleaseCapture.Call()
	return ret != 0
}

func user32PostMessageW(hwnd uintptr, msg uint32, wparam, lparam uintptr) bool {
	ret, _, _ := procPostMessageW.Call(hwnd, uintptr(msg), wparam, lparam)
	return ret != 0
}

func user32GetWindowRect(hwnd uintptr, rect *w32Rect) bool {
	ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(rect)))
	return ret != 0
}

func user32GetCursorPos(pt *w32Point) bool {
	ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(pt)))
	return ret != 0
}

// windowDPIScaleFactor converts the CSS-pixel hot-zone thickness into
// physical pixels for the window's current DPI.
func windowDPIScaleFactor(win *application.WebviewWindow) float64 {
	if win == nil {
		return 1
	}
	hwnd := win.NativeWindow()
	if hwnd == nil {
		return 1
	}
	dpi, _, _ := procGetDpiForWindow.Call(uintptr(hwnd))
	if dpi == 0 {
		return 1
	}
	return float64(dpi) / 96
}
