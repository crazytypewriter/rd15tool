package main

import (
	"fyne.io/fyne/v2/app"
	rdapp "github.com/crazytypewriter/rd15tool/internal/app"
)

func main() {
	myApp := app.New()
	myApp.Settings().SetTheme(&rdapp.MobileTheme{})
	rdapp.NewAppWindow(myApp).Window.ShowAndRun()
}
