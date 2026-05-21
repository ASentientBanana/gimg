package component

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/asentientbanana/gimg/util"
)

type SortView struct {
	// 0 initialized 1 doing 2 done
	Status int

	// Components
	FoundFilesLabel         *widget.Label
	PathLabel               *widget.Label
	DestinationDirLabel     *widget.Label
	FindDirButton           *widget.Button
	SelectDestinationButton *widget.Button
	Content                 *fyne.Container
	ListView                *widget.List
	RunButton               *widget.Button
	ScanButton              *widget.Button
	StatusLabel             *widget.Label
	ProgressCount           binding.Int
}

type SortData struct {
	dirPath         []string
	destinationPath string
	images          []util.DirScanResults
	isValid         bool
	numberOfFiles   int
}

func handleDirScan(data *SortData, dirPathIndex int) {

	entityCount := make(chan util.ScanCounterElement, 10)
	results := make(chan util.DirScanResults)
	wg := &sync.WaitGroup{}

	collectorWg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		util.ScanDirRecursiveForImageFiles(data.dirPath[dirPathIndex], wg, results, entityCount)
	}()

	go func() {
		wg.Wait()

		close(results)
		close(entityCount)
	}()

	c := 0

	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for r := range results {
			data.images = append(data.images, r)
		}
	}()

	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for range entityCount {
			c++
		}
	}()
	collectorWg.Wait()

	fmt.Println(len(data.images))

}

func NewListElement(text string, onClose func()) *fyne.Container {
	container := container.NewHBox(
		widget.NewLabel(text),
		widget.NewButton("X", onClose),
	)

	return container
}

func NewSortView(w *fyne.Window) *SortView {

	data := &SortData{
		dirPath:         []string{},
		destinationPath: "",
		isValid:         false,
		images:          []util.DirScanResults{},
		numberOfFiles:   0,
	}

	p := &SortView{
		FoundFilesLabel: widget.NewLabel("No files scanned..."),
		// StatusLabel:         widget.NewLabel(""),
		DestinationDirLabel: widget.NewLabel("No destination directory selected"),
		Status:              0,
		ProgressCount:       binding.NewInt(),
	}

	p.StatusLabel = widget.NewLabelWithData(binding.IntToString(p.ProgressCount))

	p.FindDirButton = widget.NewButton("Add a directory to scan", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri == nil {
				return
			}
			path := uri.Path()

			data.dirPath = append(data.dirPath, path)
			fmt.Println("Added the dir ", path, " to the list")
			p.ListView.Refresh()
		}, *w)
	})

	p.SelectDestinationButton = widget.NewButton("Select the destination directory", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri == nil {
				return
			}
			path := uri.Path()
			p.DestinationDirLabel.SetText(path)
			data.destinationPath = path
		}, *w)
	})

	p.ScanButton = widget.NewButton("Scan", func() {

		if len(data.dirPath) == 0 {
			dialog.ShowError(errors.New("No firectories selected."), *w)
			return
		}
		data.images = []util.DirScanResults{}
		for i := range data.dirPath {
			handleDirScan(data, i)
		}
		p.FoundFilesLabel.SetText(fmt.Sprintf("Found: %d image files.", len(data.images)))
	})

	p.ScanButton.Importance = widget.WarningImportance

	p.RunButton = widget.NewButton("Run", func() {

		if len(data.images) == 0 {
			dialog.ShowError(errors.New("No firectories selected."), *w)
			return
		}

		if data.destinationPath == "" {
			dialog.ShowError(errors.New("Select a destination folder."), *w)
			return
		}

		progressChan := make(chan int, 100)

		dMap, err := util.CreateDirMapFromIndexedData(&data.images)

		if err != nil {
			fmt.Println(err)
			return
		}

		years := []string{}

		for y := range dMap {
			years = append(years, y)
		}

		err = util.CreateFileStructure(data.destinationPath, years)

		if err != nil {
			return
		}

		go func() {
			prog := 0
			for range progressChan {
				fmt.Println("Rec prog")
				prog++

				i, err := p.ProgressCount.Get()
				if err == nil {
					p.ProgressCount.Set(i + 1)
				}

			}
		}()

		go func() {
			util.CopyFilesFromMap(dMap, data.destinationPath, progressChan)
			close(progressChan)
		}()

	})

	p.RunButton.Importance = widget.SuccessImportance

	p.ListView = widget.NewList(func() int {
		return len(data.dirPath)
	}, func() fyne.CanvasObject {
		return container.NewHBox(widget.NewLabel(""), widget.NewButton("Delete", nil))
	}, func(i widget.ListItemID, o fyne.CanvasObject) {

		cont := o.(*fyne.Container)
		label := cont.Objects[0].(*widget.Label)
		btn := cont.Objects[1].(*widget.Button)
		label.SetText(data.dirPath[i])
		btn.Text = "Delete"
		btn.OnTapped = func() {
			data.dirPath = slices.Delete(data.dirPath, i, i+1)
			p.ListView.Refresh()
		}
	})

	p.ListView.Resize(fyne.NewSize(900, p.ListView.MinSize().Height))

	p.Content = container.NewCenter(container.NewHBox(container.NewVBox(container.NewHBox(p.FindDirButton, p.ScanButton), p.FoundFilesLabel, p.SelectDestinationButton, p.DestinationDirLabel, p.RunButton, p.StatusLabel), container.New(layout.NewGridWrapLayout(fyne.NewSize(300, 400)), p.ListView)))
	return p
}
