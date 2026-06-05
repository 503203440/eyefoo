package main

import (
	"embed"
	"os"
	"path/filepath"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/icons"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	configDir := getConfigDir()
	settingsStore, _ := NewSettingsStore(configDir)
	statsStore, _ := NewStatsStore(configDir)
	timerService := NewTimerService(settingsStore, statsStore)

	app := application.New(application.Options{
		Name:        "eyefoo",
		Description: "眼睛护士 - Eye protection reminder",
		Services: []application.Service{
			application.NewService(settingsStore),
			application.NewService(timerService),
			application.NewService(statsStore),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	// system tray
	tray := app.SystemTray.New()
	tray.SetTooltip("眼睛护士 Eyefoo")
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icons.SystrayMacTemplate)
	}
	trayMenu := app.NewMenu()
	trayMenu.Add("设置").OnClick(func(ctx *application.Context) {
		timerService.ShowSettings()
	})
	trayMenu.Add("开始/暂停").OnClick(func(ctx *application.Context) {
		timerService.Toggle()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("退出").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	tray.SetMenu(trayMenu)

	// settings window
	settingsWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:   "眼睛护士 Eyefoo",
		Width:   420,
		Height:  580,
		Hidden:  true,
		Frameless: true,
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
			Backdrop: application.MacBackdropTranslucent,
		},
		BackgroundColour: application.NewRGB(30, 30, 30),
		URL:              "/",
	})
	settingsWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		settingsWindow.Hide()
		e.Cancel()
	})
	settingsWindow.RegisterHook(events.Common.WindowLostFocus, func(e *application.WindowEvent) {
		state := timerService.GetState()
		if state.Phase != 2 {
			settingsWindow.Hide()
		}
	})

	tray.AttachWindow(settingsWindow).WindowOffset(2)
	timerService.setWindow(settingsWindow, app)

	// start timer on launch
	timerService.Start()

	err := app.Run()
	if err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}

func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".config", "eyefoo")
	os.MkdirAll(dir, 0755)
	return dir
}
