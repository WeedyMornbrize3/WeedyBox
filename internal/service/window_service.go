package service

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

type WindowService struct {
	Window *application.WebviewWindow
}

// 窗口控制方法：最小化

func (w *WindowService) MinimizeWindow() {
	if w.Window != nil {
		w.Window.Minimise()
	}
}

// 窗口控制方法：最大化/还原
func (w *WindowService) ToggleMaximizeWindow() {
	if w.Window == nil {
		return
	}
	if w.Window.IsMaximised() {
		w.Window.UnMaximise()
	} else {
		w.Window.Maximise()
	}
}

// 窗口控制方法：关闭
func (w *WindowService) CloseWindow() {
	if w.Window != nil {
		w.Window.Close()
	}
}

// 窗口控制方法：获取窗口状态（让前端知道当前是否最大化）
func (w *WindowService) IsWindowMaximised() bool {
	if w.Window != nil {
		return w.Window.IsMaximised()
	}
	return false
}
