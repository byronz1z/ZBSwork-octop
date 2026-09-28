//go:build !windows

package main

import "github.com/wailsapp/wails/v3/pkg/application"

const w32WMNCLButtonDown = 0x00A1 // WM_NCLBUTTONDOWN

// w32HitTestCodes mirrors wails' edgeMap; on non-Windows platforms the resize
// hook is a no-op, but the table keeps window_chrome.go compiling everywhere.
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

func user32ReleaseCapture() bool { return false }

func user32PostMessageW(_ uintptr, _ uint32, _, _ uintptr) bool { return false }

func user32GetWindowRect(_ uintptr, _ *w32Rect) bool { return false }

func user32GetCursorPos(_ *w32Point) bool { return false }

func windowDPIScaleFactor(_ *application.WebviewWindow) float64 { return 1 }
