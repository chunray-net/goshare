package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	a := app.NewWithID("chunray.net")
	w := a.NewWindow("GoShare Fyne 客户端")
	w.Resize(fyne.NewSize(360, 480))

	//setupSystray(w, a)

	ui := NewUI(a, w)
	ui.Run()
}
