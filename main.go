package main

import (
	"embed"

	"WeedyBox/internal/database"
	"WeedyBox/internal/service"
	"WeedyBox/internal/service/getMeta"
	"WeedyBox/internal/service/scan"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/sqlite"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

// 托盘图标：复用构建用的多尺寸 ICO，并在编译期嵌入二进制，
// 这样运行时不依赖工作目录（应用可能从 bin/ 启动）。
//
//go:embed build/windows/icon.ico
var trayIcon []byte

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[string]("time")
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {

	//注册服务
	windowService := &service.WindowService{}

	sqliteService := sqlite.NewWithConfig(&sqlite.Config{
		DBSource: "WeedyBoxDB.db",
	})
	dbService := &database.SQLiteService{
		DB: sqliteService,
	}
	todoService := &service.TodoService{
		DB: dbService,
	}

	// 元数据抓取服务：调试页（/testing）用它「每次真去 DLsite 抓一遍」核对解析结果。
	// 封面图片会落到工作目录下的相对路径 preImg/，并把落地位置记进游戏库。
	// 搜索结果按 DLsite 分页逐页取（每页固定 30 条，per_page 改大无效），page 参数由前端传。
	metaService := &getmeta.MetaApiService{DB: dbService}

	// 游戏库服务：作品库（/works）与游戏库（/library）用。
	// 与 metaService 的区别是**缓存优先** —— 库里有的作品不再联网。
	libraryService := &service.LibraryService{
		DB:   dbService,
		Meta: metaService,
	}

	// 本地扫描服务：游戏库「扫描添加」用 —— Steam 库里的游戏、或任意本地游戏文件夹。
	// 只读磁盘、不联网；扫到的名字交给前端去 SearchWorks 找作品。
	scanService := &scan.ScanService{}

	// 系统对话框服务：目前只有「选一个游戏文件夹」这一个用途。
	dialogService := &service.DialogService{}

	applicationServices := []application.Service{
		application.NewService(windowService),
		application.NewService(todoService),
		application.NewService(dbService),
		application.NewService(metaService),
		application.NewService(libraryService),
		application.NewService(scanService),
		application.NewService(dialogService),
	}

	app := application.New(application.Options{
		Name:        "WeedyBox",
		Description: "A demo of using raw HTML & CSS",
		Services:    applicationServices,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "WeedyBox",
		// Window sized to the golden ratio (1000 / 618 ≈ 1.618).
		Width:     1200,
		Height:    741,
		MinWidth:  900,
		MinHeight: 556,
		Frameless: true, //无边框
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(255, 255, 255),
		URL:              "/",
	})

	windowService.Window = win

	// 系统托盘：窗口「隐藏到后台」之后唯一的唤回入口，必须在这里创建。
	// 菜单里额外给一个明确的「退出程序」，否则隐藏后只能靠任务管理器结束。
	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip("WeedyBox")
	tray.AttachWindow(win)

	// 左键点击托盘的语义：**只负责唤回，不负责隐藏**。
	// 不能用 Wails 的默认 ToggleWindow —— 它在窗口已显示时会把它隐藏，
	// 那样「点托盘把小窗口叫出来」之后手滑再点一下又没了。
	// 隐藏统一走标题栏 X（或设置里的「隐藏到托盘」）。
	tray.OnClick(func() {
		if win.IsVisible() {
			return
		}
		_ = tray.PositionWindow(win, 0) // 定位到托盘图标附近，0 表示默认间距
		// Show() 对「隐藏」的窗口是唤回；对「已最小化」的窗口则不生效
		// （Windows 下最小化的窗口需要 UnMinimise），所以两步都做，幂等。
		win.Show()
		win.UnMinimise()
		win.Focus()
	})

	trayMenu := application.NewMenu()
	trayMenu.Add("显示主窗口").OnClick(func(_ *application.Context) {
		win.UnMinimise()
		win.Show().Focus()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("退出程序").OnClick(func(_ *application.Context) {
		app.Quit()
	})
	tray.SetMenu(trayMenu)

	// Create a goroutine that emits an event containing the current time every second.
	// The frontend can listen to this event and update the UI accordingly.
	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			app.Event.Emit("time", now)
			time.Sleep(time.Second)
		}
	}()

	// 退出前清空搜索缩略图缓存：cache/ 只是"省一次网络请求"的临时图，
	// 下次搜索会自动重下，没必要长期占盘（preImg/ 是入库存档，不能碰）。
	app.OnShutdown(func() {
		if err := getmeta.ClearCacheDir(); err != nil {
			log.Printf("清理缩略图缓存失败: %v", err)
		}
	})

	// Run the application. This blocks until the application has been exited.
	err := app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
