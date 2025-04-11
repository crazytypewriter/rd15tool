package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	rdapp "github.com/crazytypewriter/rd15tool/internal/app"
	"runtime"
)

func main() {
	myApp := app.New()
	myApp.Settings().SetTheme(LoadPlatformTheme())
	rdapp.NewAppWindow(myApp).Window.ShowAndRun()
}

func LoadPlatformTheme() fyne.Theme {
	switch runtime.GOOS {
	case "darwin":
		return &rdapp.MacTheme{}
	case "windows":
		return &rdapp.WindowsTheme{}
	default:
		return &rdapp.MobileTheme{}
	}
}
