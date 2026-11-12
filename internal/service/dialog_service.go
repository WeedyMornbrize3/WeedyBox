// internal/service/dialog_service.go
//
// 系统文件对话框。单独一个服务而不是塞进 WindowService：
// 「窗口怎么摆」与「问用户要一个路径」是两件事，混在一起以后加保存框/
// 消息框又得改窗口服务的语义。
//
// 用法（Wails v3 beta）：app.Dialog.OpenFile() 返回一个可链式配置的对话框，
// PromptForSingleSelection 内部走 InvokeSync 切到主线程再弹，所以从服务
// goroutine 里直接调用是安全的；用户取消时返回空串 + nil（不是错误）。
package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// DialogService 暴露系统对话框给前端。
type DialogService struct{}

// PickDirectory 打开系统「选择文件夹」对话框，返回用户选中的绝对路径。
//
// 空标题用默认文案。用户取消 → 空串 + nil：取消不是错误，前端据此什么都不做；
// 只有应用实例没起来、或对话框创建失败才报错。
func (s *DialogService) PickDirectory(title string) (string, error) {
	app := application.Get()
	if app == nil {
		return "", errors.New("应用实例未就绪，无法打开文件夹选择框")
	}

	dialog := app.Dialog.OpenFile().
		CanChooseFiles(false).
		CanChooseDirectories(true).
		AllowsOtherFileTypes(false)
	if trimmed := strings.TrimSpace(title); trimmed != "" {
		dialog.SetTitle(trimmed)
	}
	// 挂在当前窗口上：对话框居中于主窗口，而不是飘到桌面左上角
	if window := app.Window.Current(); window != nil {
		dialog.AttachToWindow(window)
	}

	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("打开文件夹选择框失败: %w", err)
	}
	return strings.TrimSpace(path), nil
}
