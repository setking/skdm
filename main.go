package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"time"

	"changeme/backed/api/apiserver"
	"changeme/backed/pkg/store"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var trayIcon []byte

// forceQuit is set to true when the user selects "退出程序" from the tray menu,
// allowing the window close hook to let the window actually close.
var forceQuit bool

func init() {
	// Register custom events for download status/progress updates.
	// "download-update" carries a full DownloadRecord for incremental UI sync.
	// "download-removed" carries the GID string of a removed download.
	application.RegisterEvent[store.DownloadRecord]("download-update")
	application.RegisterEvent[string]("download-removed")

	// "tray-new-task" is emitted when the user clicks "新建任务" in the tray menu.
	// The frontend listens for this to show the download dialog.
	application.RegisterEvent[string]("tray-new-task")

	// Legacy time event from template
	application.RegisterEvent[string]("time")
}

// openDownloadPopup creates (or reuses) a small centered popup window
// that renders just the download dialog at /tray-task.
func openDownloadPopup(app *application.App) {
	const popupName = "skdm-download-dialog"
	// Reuse existing popup if it already exists
	if popup, ok := app.Window.GetByName(popupName); ok {
		popup.Center()
		popup.Show().Focus()
		return
	}
	popup := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:                      popupName,
		Title:                     "新建下载任务",
		Width:                     600,
		Height:                    450,
		URL:                       "/#/tray-task",
		Frameless:                 true,
		DisableResize:             true,
		BackgroundColour:          application.NewRGB(27, 38, 54),
		DefaultContextMenuDisabled: true,
	})
	popup.Center()
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	cacheDir, _ := os.UserCacheDir()
	dbPath := filepath.Join(cacheDir, "skdm", "skdm.db")

	app := application.New(application.Options{
		Name:        "skdm",
		Description: "skdm下载器",
		Services: []application.Service{
			application.NewService(apiserver.NewAria2Service(dbPath)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "skdm",
		Width:     1024,
		Height:    680,
		MinWidth:  1024,
		MinHeight: 680,
		Frameless: true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour:          application.NewRGB(27, 38, 54),
		DefaultContextMenuDisabled: true,
		URL:                       "/",
	})

	// Intercept window close to hide instead of destroy.
	// This allows the app to stay alive in the system tray.
	mainWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if forceQuit {
			return // allow actual close when quitting from tray menu
		}
		event.Cancel()
		mainWindow.Hide()
	})

	// Set up system tray
	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip("SKDM - 下载管理器")
	tray.AttachWindow(mainWindow)

	// Override left-click to toggle window centered on screen
	// (default AttachWindow behavior positions near tray icon, we want centered)
	tray.OnClick(func() {
		if mainWindow.IsVisible() {
			mainWindow.Hide()
		} else {
			mainWindow.Center()
			mainWindow.Show().Focus()
		}
	})

	// Build right-click tray menu
	trayMenu := app.NewMenu()
	trayMenu.Add("新建任务").OnClick(func(ctx *application.Context) {
		if mainWindow.IsVisible() {
			// Main window visible: show dialog inside it
			app.Event.Emit("tray-new-task", "")
		} else {
			// Main window hidden: open a standalone popup dialog centered on screen
			openDownloadPopup(app)
		}
	})
	trayMenu.Add("打开主面板").OnClick(func(ctx *application.Context) {
		if !mainWindow.IsVisible() {
			mainWindow.Center()
		}
		mainWindow.Show().Focus()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("退出程序").OnClick(func(ctx *application.Context) {
		forceQuit = true
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

	// Run the application. This blocks until the application has been exited.
	err := app.Run()
	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
