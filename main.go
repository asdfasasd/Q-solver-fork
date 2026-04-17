package main

import (
	"embed"
	"os"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Windows 专用环境变量
	if runtime.GOOS == "windows" {
		os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGS", "--disable-gpu")
	}

	app := NewApp()
	err := wails.Run(&options.App{
		Title:     "",
		Width:     1024,
		Height:    768,
		MinWidth:  840,
		MinHeight: 700,
		MaxWidth:  1024,
		MaxHeight: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		AlwaysOnTop:      true,
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.None,
			WebviewBrowserPath:   "",
			Theme:                windows.SystemDefault,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHidden(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false, // 禁用系统毛玻璃效果，避免白色遮罩
			About: &mac.AboutInfo{
				Title:   "Q-Solver",
				Message: "AI 笔试助手",
			},
		},
		OnShutdown: app.OnShutdown,
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
