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
	// w.Resize(fyne.NewSize(800, 600))

	// Views (Structs containing the components)
	sortView := component.NewSortView(&w)

	sortTab := container.NewTabItem("Sort", sortView.Content)

	// dedupTab := container.NewTabItem("Dedup", widget.NewButton("Test", func() {
	// fmt.Println("test")

	// }))
	tabs := container.NewAppTabs(sortTab)

	w.SetContent(tabs)
	w.ShowAndRun()
	cleanup()
}

func cleanup() {
	fmt.Println("Done !!")
}
