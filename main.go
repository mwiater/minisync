package main

import (
	"embed"

	"github.com/wailsapp/wails/v2/pkg/application"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// assets contains the embedded frontend assets for the application.
// These assets are served by the Wails application during runtime.
//
//go:embed frontend/src
var assets embed.FS

// icon contains the embedded application icon in PNG format.
// This icon is used in the application window and in the about dialog on macOS.
//
//go:embed build/appicon.png
var icon []byte

// main is the entry point of the Minisync application. It creates and configures the
// Wails application with platform-specific options and then runs the application.
func main() {
	// Create an instance of the app structure
	App := NewApp()

	// Create application with options
	app := application.NewWithOptions(&options.App{
		Title:              "MiniSync",
		Width:              1024,
		Height:             550,
		MinWidth:           1024,
		MinHeight:          550,
		DisableResize:      false,
		Fullscreen:         false,
		Frameless:          false,
		StartHidden:        false,
		HideWindowOnClose:  false,
		BackgroundColour:   &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		Assets:             assets,
		Menu:               nil,
		Logger:             nil,
		LogLevel:           logger.INFO,
		LogLevelProduction: logger.INFO,
		OnStartup:          App.startup,
		OnDomReady:         App.domReady,
		OnBeforeClose:      App.beforeClose,
		OnShutdown:         App.shutdown,
		WindowStartState:   options.Normal,
		Bind: []interface{}{
			App,
		},
		Debug: options.Debug{
			OpenInspectorOnStartup: true,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			WebviewUserDataPath:  "",
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "MiniSync",
				Message: "",
				Icon:    icon,
			},
		},
	})

	// Run the application
	app.Run()
}
