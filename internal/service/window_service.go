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

// 窗口控制方法：隐藏到后台（不占任务栏）。
// ⚠️ 与 MinimizeWindow 的区别：最小化仍留在任务栏，隐藏则完全不可见，
// 只能靠系统托盘图标唤回——所以托盘必须已创建（见 main.go 的 setupSystemTray）。
func (w *WindowService) HideWindow() {
	if w.Window != nil {
		w.Window.Hide()
	}
}

// 窗口控制方法：从后台恢复显示并聚焦
func (w *WindowService) ShowWindow() {
	if w.Window != nil {
		w.Window.Show().Focus()
	}
}

// 窗口控制方法：当前是否可见（前端可据此判断自己是否处于后台）
func (w *WindowService) IsWindowVisible() bool {
	if w.Window != nil {
		return w.Window.IsVisible()
	}
	return false
}
