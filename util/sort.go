package util

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/rwcarlsen/goexif/exif"
)

type GetDirResult struct {
	Files []string
	Dirs  []string
}

type ContentParams struct {
	Path    string
	Filter  string
	FileSet *map[string]struct{}
}

type ExifData struct {
	DataString string
	Year       string
	Month      string
}

type ScanCounterElement struct {
	Type string
}

type DirScanResults struct {
	Path            string
	Data            ExifData
	Name            string
	BackupPath      string
	ExifUnavailable bool
}

var FILE_SET_WHITELIST = make(map[string]struct{})

func InitFileTypes() {
	// Init file types
	FILE_SET_WHITELIST["jpg"] = struct{}{}
	FILE_SET_WHITELIST["jpeg"] = struct{}{}
	FILE_SET_WHITELIST["png"] = struct{}{}
}

func GetFileTypeFromName(name string) string {
	if len(name) == 0 {
		return ""
	}
	ext := strings.ToLower(path.Ext(name))
	if len(ext) < 1 {
		return ""
	}
	return ext[1:]
}

// filter can be nil, dir or file
func GetDirContents(cp ContentParams) GetDirResult {
	contents := GetDirResult{}
	_readContent, err := os.ReadDir(cp.Path)

	isFile := cp.Filter == "f"
	isDir := cp.Filter == "d"

	if err != nil {
		return contents
	}

	for _, v := range _readContent {
		if v.IsDir() {
			if isFile {
				continue
			}
			contents.Dirs = append(contents.Dirs, path.Join(cp.Path, v.Name()))
		} else {
			if isDir {
				continue
			}
			if cp.FileSet == nil {
				contents.Files = append(contents.Files, path.Join(cp.Path, v.Name()))
			} else {
				_type := GetFileTypeFromName(v.Name())
				// fmt.Println(_type)
				if _, ok := (*cp.FileSet)[_type]; ok {
					contents.Files = append(contents.Files, path.Join(cp.Path, v.Name()))
				}
			}
		}
	}
	return contents
}

func CreateMonthDirectories(_path string) {
	for i := range 12 {
		name := i + 1
		dirPath := path.Join(_path, strconv.Itoa(name))
		d, err := os.Stat(dirPath)
		if err != nil {
			break
		}
		if d.IsDir() {
			continue
		}
		err = os.Mkdir(dirPath, os.ModeDir)
	}
}

func ExtractCreationData(filepath string) error {
	// if no exif data available, use file created date.

	if runtime.GOOS == "windows" {
		// Todo(petar): maybe ignore windows
		_, err := os.Stat(filepath)
		if err != nil {
			return err
		}
		//Todo(petar): Continue the windows handler
		// d := fi.Sys().(*syscall.)
		// if !ok {
		// 	return time.Time{}, fmt.Errorf("not a Windows file attribute")
		// }
	} else {

	}
	return nil
}

func ScanDirRecursiveForImageFiles(path string, wg *sync.WaitGroup, baseDir string, result chan<- DirScanResults, counter chan<- ScanCounterElement) {
	defer wg.Done()
	contents := GetDirContents(ContentParams{
		Filter:  "",
		Path:    path,
		FileSet: &FILE_SET_WHITELIST,
	})

	for _, d := range contents.Dirs {
		counter <- ScanCounterElement{Type: "dir"}
		wg.Add(1)
		newBaseDir := filepath.Join(baseDir, filepath.Base(d))
		go ScanDirRecursiveForImageFiles(d, wg, newBaseDir, result, counter)
	}
	for _, f := range contents.Files {

		fData, err := ExtractExifDate(f)
		if err != nil || fData.DataString == " " || fData.DataString == "" {
			fName := filepath.Base(f)

			counter <- ScanCounterElement{Type: "file"}
			result <- DirScanResults{Path: f, Data: fData, ExifUnavailable: true, Name: fName, BackupPath: baseDir}
			continue
		}

		counter <- ScanCounterElement{Type: "file"}
		result <- DirScanResults{Path: f, Data: fData, ExifUnavailable: false, Name: "", BackupPath: ""}
	}
}

func UniquePath(dir, filename string) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	candidate := filepath.Join(dir, filename)
	for i := 1; ; i++ {
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate // no collision, use it
		}
		candidate = filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
	}
}

func ExtractExifDate(f string) (ExifData, error) {
	const formatString = "2006:01:02 15:04:05"
	i, err := os.Open(f)
	data := ExifData{}
	if err != nil {
		return data, err
	}
	defer i.Close()
	ex, err := exif.Decode(i)
	if err != nil {
		//Todo(peata): handle no exif data case
		return data, errors.New("NO EXIF DATA AVAILABLE.")
	}

	t, err := ex.DateTime()
	if err != nil {
		return data, nil
	}

	data.Year = strconv.Itoa(t.Year())
	data.Month = strconv.Itoa(int(t.Month()))
	data.DataString = strings.Replace(t.Format(formatString), " ", "_", 1)

	return data, nil
}

type DirMap map[string]map[string][]DirScanResults

func CreateDirMapFromIndexedData(sr *[]DirScanResults) (DirMap, error) {
	dirMap := make(DirMap)
	for _, v := range *sr {
		// Checking for files with unavailable exif data to separate them.
		if v.ExifUnavailable {
			_, ok := dirMap["unknown"]
			if !ok {
				fmt.Println("Creating unlnown dir map entry")
				dirMap["unknown"] = make(map[string][]DirScanResults)
				dirMap["unknown"]["unknown"] = []DirScanResults{}
			}
			dirMap["unknown"]["unknown"] = append(dirMap["unknown"]["unknown"], v)
			continue
		}

		_, ok := dirMap[v.Data.Year]
		if !ok {
			dirMap[v.Data.Year] = make(map[string][]DirScanResults)
		}

		_, ok = dirMap[v.Data.Year][v.Data.Month]

		if ok {
			dirMap[v.Data.Year][v.Data.Month] = append(dirMap[v.Data.Year][v.Data.Month], v)

		} else {
			dirMap[v.Data.Year][v.Data.Month] = []DirScanResults{}
			dirMap[v.Data.Year][v.Data.Month] = append(dirMap[v.Data.Year][v.Data.Month], v)
		}
	}
	return dirMap, nil
}

func DumpToFile(data string, p string) error {
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	_, err = f.WriteString(data)
	return err
}

func CreateFileStructure(dirPath string, dirMap *DirMap) error {

	for year := range *dirMap {
		yearPath := path.Join(dirPath, year)
		e, err := os.Stat(yearPath)
		if err == nil && e.IsDir() {
			continue
		}

		err = os.Mkdir(yearPath, 0755)

		if err != nil {
			return err
		}

		if year == "unknown" {
			for _, unknownFile := range (*dirMap)[year][year] {
				os.MkdirAll(filepath.Join(yearPath, unknownFile.BackupPath), 0755)
			}
			continue
		}
		//Create month directories
		for m := range (*dirMap)[year] {
			os.Mkdir(path.Join(yearPath, m), 0755)
		}

	}
	return nil
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

func CopyFilesFromMap(_map *DirMap, destination string) (int, error) {
	wg := &sync.WaitGroup{}
	files := make(chan bool, 10)

	// Create date directories
	if err := CreateFileStructure(destination, _map); err != nil {
		return 0, err
	}

	for y := range *_map {

		year, ok := (*_map)[y]

		if !ok {
			continue
		}

		if y == "unknown" {
			wg.Add(1)
			go func(_wg *sync.WaitGroup, fChan chan<- bool) {

				for _, elem := range (*_map)[y] {

					for _, e := range elem {
						dst := path.Join(destination, "unknown", e.BackupPath, e.Name)
						_ = copyFile(e.Path, dst)
						fChan <- true
					}
				}
				_wg.Done()
			}(wg, files)
			continue
		}

		for _, month := range year {
			wg.Add(1)
			go func(_wg *sync.WaitGroup, fChan chan<- bool) {
				for _, d := range month {
					dst := path.Join(destination, d.Data.Year, d.Data.Month, d.Data.DataString+path.Ext(d.Path))
					err := copyFile(d.Path, dst)

					if err != nil {
						fChan <- false
					} else {
						fChan <- true
					}
				}
				_wg.Done()

			}(wg, files)
		}
	}

	go func() {
		wg.Wait()
		close(files)
	}()

	filesTransfered := 0
	filesFailed := 0

	for f := range files {
		if f {
			filesTransfered++
		} else {
			filesFailed++
		}
	}
	fmt.Printf("\n Transfered %d files  \n", filesTransfered)
	return filesTransfered, nil
}
