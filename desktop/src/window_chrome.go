package main

import (
	"math"
	"strconv"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// desktopDragRegionClass must stay in sync with dashboard DESKTOP_DRAG_REGION_CLASS.
// Frameless moving uses CSS `--wails-draggable: drag` plus this injected starter:
// the remote dashboard origin never loads Wails `/wails/runtime.js`.
// clientY <= 32 must match dashboard DESKTOP_TITLEBAR_DRAG_HEIGHT.
const desktopDragRegionClass = "octop-desktop-drag"

// resizeBorderSize is the transparent edge/corner hot-zone thickness in CSS
// pixels. The dashboard never loads Wails `/wails/runtime.js`, so its built-in
// frameless resize detection never runs; the overlay below arms the same
// wails:resize:<edge> protocol instead.
const resizeBorderSize = 6

func dragOverlayJS() string {
	return `(function(){
		if (!document.body || !window._wails || typeof window._wails.invoke !== 'function') return;
		if (document.documentElement.dataset.octopDragReady === '1') return;
		document.documentElement.dataset.octopDragReady = '1';
		var armed = false, startX = 0, startY = 0;
		var noDrag = 'button, a, input, textarea, select, [role="button"], [role="menuitem"], [data-octop-no-drag], .octop-desktop-no-drag';
		function targetEl(t) {
			if (t && t.nodeType === 1) return t;
			return t && t.parentElement ? t.parentElement : null;
		}
		function shouldArm(event) {
			if (event.button !== 0) return false;
			var el = targetEl(event.target);
			if (!el || !el.closest) return false;
			if (el.closest(noDrag)) return false;
			var value = window.getComputedStyle(el).getPropertyValue('--wails-draggable').trim();
			if (value === 'no-drag') return false;
			if (value === 'drag') return true;
			return event.clientY <= 32;
		}
		window.addEventListener('mousedown', function(event) {
			if (!shouldArm(event)) return;
			armed = true;
			startX = event.screenX;
			startY = event.screenY;
		}, true);
		window.addEventListener('mousemove', function(event) {
			if (!armed) return;
			if (Math.abs(event.screenX - startX) < 3 && Math.abs(event.screenY - startY) < 3) return;
			armed = false;
			window._wails.invoke('wails:drag');
		}, true);
		window.addEventListener('mouseup', function() { armed = false; }, true);
		window.addEventListener('dblclick', function(event) {
			if (!shouldArm(event)) return;
			window._wails.invoke('wails:drag:doubleclick');
		}, true);
	})();`
}

// resizeOverlayJS installs transparent edge/corner hot zones on the loaded
// page so a frameless window can be resized by dragging its borders. The
// mousedown handler invokes wails:resize:<edge>, which the WindowStartResize
// hook turns into a native WM_NCLBUTTONDOWN resize loop.
func resizeOverlayJS() string {
	return `(function(){
		if (!document.body || !window._wails || typeof window._wails.invoke !== 'function') return;
		if (document.documentElement.dataset.octopResizeReady === '1') return;
		document.documentElement.dataset.octopResizeReady = '1';
		var border = ` + strconv.Itoa(resizeBorderSize) + `;
		var style = document.createElement('style');
		style.textContent = '.octop-desktop-resize{position:fixed;z-index:2147483647;box-sizing:border-box;background:transparent;}';
		document.head.appendChild(style);
		var zones = [
			['n', 'top:0;left:' + border + 'px;right:' + border + 'px;height:' + border + 'px;cursor:ns-resize;'],
			['s', 'bottom:0;left:' + border + 'px;right:' + border + 'px;height:' + border + 'px;cursor:ns-resize;'],
			['w', 'left:0;top:' + border + 'px;bottom:' + border + 'px;width:' + border + 'px;cursor:ew-resize;'],
			['e', 'right:0;top:' + border + 'px;bottom:' + border + 'px;width:' + border + 'px;cursor:ew-resize;'],
			['nw', 'top:0;left:0;width:' + border + 'px;height:' + border + 'px;cursor:nwse-resize;'],
			['ne', 'top:0;right:0;width:' + border + 'px;height:' + border + 'px;cursor:nesw-resize;'],
			['sw', 'bottom:0;left:0;width:' + border + 'px;height:' + border + 'px;cursor:nesw-resize;'],
			['se', 'bottom:0;right:0;width:' + border + 'px;height:' + border + 'px;cursor:nwse-resize;']
		];
		zones.forEach(function(zone) {
			var el = document.createElement('div');
			el.className = 'octop-desktop-resize';
			el.dataset.octopResizeEdge = zone[0];
			el.style.cssText = zone[1];
			document.body.appendChild(el);
		});
		document.body.addEventListener('mousedown', function(event) {
			if (event.button !== 0) return;
			var el = event.target && event.target.closest ? event.target.closest('.octop-desktop-resize') : null;
			if (!el) return;
			event.preventDefault();
			event.stopPropagation();
			window._wails.invoke('wails:resize:' + el.dataset.octopResizeEdge + '-resize');
		}, true);
	})();`
}

// startWindowResize posts the native WM_NCLBUTTONDOWN message that starts a
// window resize for the given edge, mirroring wails' own frameless resize
// entry point (see windowsWebviewWindow.startResize).
func (a *App) startWindowResize(win *application.WebviewWindow) {
	if win == nil {
		return
	}
	hwnd := win.NativeWindow()
	if hwnd == nil {
		return
	}
	edge, ok := resizeEdgeFromCursor(win)
	if !ok {
		return
	}
	if !user32ReleaseCapture() {
		return
	}
	user32PostMessageW(uintptr(hwnd), w32WMNCLButtonDown, w32HitTestCodes[edge], 0)
}

// resizeEdgeFromCursor derives the resize direction from the current pointer
// position relative to the window rect. It must stay in sync with the hot
// zones injected by resizeOverlayJS (resizeBorderSize CSS pixels).
func resizeEdgeFromCursor(win *application.WebviewWindow) (string, bool) {
	hwnd := win.NativeWindow()
	if hwnd == nil {
		return "", false
	}
	var rect w32Rect
	if !user32GetWindowRect(uintptr(hwnd), &rect) {
		return "", false
	}
	var pt w32Point
	if !user32GetCursorPos(&pt) {
		return "", false
	}
	width := int(rect.right - rect.left)
	height := int(rect.bottom - rect.top)
	if width <= 0 || height <= 0 {
		return "", false
	}
	x := int(pt.x - rect.left)
	y := int(pt.y - rect.top)
	if x < 0 || y < 0 || x >= width || y >= height {
		return "", false
	}
	// Ceiling keeps the hottest inner pixel of the CSS zone inside the
	// native border at fractional scale factors (e.g. 125% -> 6px = 7.5px).
	border := int(math.Ceil(resizeBorderSize * windowDPIScaleFactor(win)))
	if border < 1 {
		border = 1
	}
	left := x < border
	right := x >= width-border
	top := y < border
	bottom := y >= height-border
	switch {
	case top && left:
		return "nw-resize", true
	case top && right:
		return "ne-resize", true
	case bottom && left:
		return "sw-resize", true
	case bottom && right:
		return "se-resize", true
	case top:
		return "n-resize", true
	case bottom:
		return "s-resize", true
	case left:
		return "w-resize", true
	case right:
		return "e-resize", true
	}
	return "", false
}
