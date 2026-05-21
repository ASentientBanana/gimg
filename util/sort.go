package util

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

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
	// dir file
	Type string
}

type DirScanResults struct {
	Path string
	Data ExifData
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

func CheckForMonthFolders(contents []string) (bool, error) {

	_d := map[string]bool{
		"1":  false,
		"2":  false,
		"3":  false,
		"4":  false,
		"5":  false,
		"6":  false,
		"7":  false,
		"8":  false,
		"9":  false,
		"10": false,
		"11": false,
		"12": false,
	}

	for _, f := range contents {

		val, ok := _d[f]

		if !ok {
			continue
		}
		if val == true {
			return false, errors.New("conflicting dir names")
		}

		_d[f] = true

	}
	count := 0
	for i := range 12 {
		val, ok := _d[strconv.Itoa(i)]
		if ok && val {
			count++
		}
	}
	// Test
	return count == 12, nil
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

func ScanDirRecursiveForImageFiles(path string, wg *sync.WaitGroup, result chan<- DirScanResults, counter chan<- ScanCounterElement) {
	defer wg.Done()
	contents := GetDirContents(ContentParams{
		Filter:  "",
		Path:    path,
		FileSet: &FILE_SET_WHITELIST,
	})

	for _, d := range contents.Dirs {
		counter <- ScanCounterElement{Type: "dir"}
		wg.Add(1)
		go ScanDirRecursiveForImageFiles(d, wg, result, counter)
	}
	for _, f := range contents.Files {
		// log.Printf("Working on %s", f)
		fData, err := ExtractExifDate(f)
		// fmt.Printf("File: %s", f)
		if err != nil {
			fmt.Println("Miss")
			fmt.Println(err.Error())
			continue
		}
		counter <- ScanCounterElement{Type: "file"}
		result <- DirScanResults{Path: f, Data: fData}
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
		info, err2 := i.Stat()
		if err2 != nil {
			fmt.Println(err2.Error())
			return data, err
		}
		modTime := info.ModTime()
		data.Year = strconv.Itoa(modTime.Year())
		data.Month = strconv.Itoa(int(modTime.Month()))
		data.DataString = strings.Replace(modTime.Format(formatString), " ", "_", 1)
		return data, nil
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
		_, ok := dirMap[v.Data.Year]
		if !ok {
			// Populate months
			for i := range 12 {
				dirMap[v.Data.Year] = make(map[string][]DirScanResults)
				dirMap[v.Data.Year][strconv.Itoa(i+1)] = []DirScanResults{}
			}
		}

		dirMap[v.Data.Year][v.Data.Month] = append(dirMap[v.Data.Year][v.Data.Month], v)

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

func CreateFileStructure(dirPath string, years []string) error {
	fmt.Println(dirPath)
	for _, year := range years {
		yearPath := path.Join(dirPath, year)
		e, err := os.Stat(yearPath)
		if err == nil && e.IsDir() {
			continue
		}
		err = os.Mkdir(yearPath, 0755)
		if err != nil {
			return err
		}
		for i := range 12 {
			err := os.Mkdir(path.Join(yearPath, strconv.Itoa(i+1)), 0755)
			if err != nil {
				return err
			}
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

func CopyFilesFromMap(_map DirMap, destination string, progressChan chan<- int) (int, error) {
	wg := &sync.WaitGroup{}
	files := make(chan bool, 10)

	// Create date dirs
	years := []string{}

	for y := range _map {
		years = append(years, y)
	}

	if err := CreateFileStructure(destination, years); err != nil {
		return 0, err
	}

	for y := range _map {
		year, ok := _map[y]
		if !ok {
			continue
		}
		for _, month := range year {
			wg.Add(1)
			go func(_wg *sync.WaitGroup, fChan chan<- bool) {
				for _, d := range month {
					// Do smtn
					dst := path.Join(destination, d.Data.Year, d.Data.Month, d.Data.DataString+path.Ext(d.Path))
					// fmt.Println(dst)
					err := copyFile(d.Path, dst)
					time.Sleep(3000 * time.Microsecond)

					progressChan <- 1
					fmt.Println("Send prog")
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
