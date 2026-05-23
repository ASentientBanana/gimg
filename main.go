package main

import (
	"fmt"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/asentientbanana/gimg/component"
	"github.com/asentientbanana/gimg/util"
)

func main() {

	// Init
	util.InitFileTypes()

	a := app.New()
	w := a.NewWindow("")

	// Views (Structs containing the components)
	sortView := component.NewSortView(&w)

	sortTab := container.NewTabItem("Sort", sortView.Content)

	tabs := container.NewAppTabs(sortTab)

	w.SetContent(tabs)
	w.ShowAndRun()
	cleanup()
}

func cleanup() {
	fmt.Println("Done !!")
}
