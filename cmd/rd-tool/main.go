package main

import (
	"fyne.io/fyne/v2/app"
	rdapp "io.rd15.tool/internal/app"
)

func main() {
	myApp := app.New()
	rdapp.NewAppWindow(myApp).Window.ShowAndRun()
}
