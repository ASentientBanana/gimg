package util

// func BenchmarkDirScanUnBuffered(b *testing.B) {
// 	path := "/home/petar/Pictures"
// 	for i := 0; i < b.N; i++ {
// 		results := make(chan DirScanResults)
// 		wg := &sync.WaitGroup{}
// 		wg.Add(1)
// 		go func(dir string) {
// 			defer wg.Done()
// 			ScanDirRecursiveForImageFiles(dir, wg, results, nil)
// 		}(path)
// 		go func() {
// 			wg.Wait()
// 			close(results)
// 		}()

// 		ff := []string{}

// 		for r := range results {
// 			ff = append(ff, r.Path)
// 		}
// 	}
// }
//
// func BenchmarkDirScanBuffered(b *testing.B) {
// 	path := "/home/petar/Pictures"
// 	InitFileTypes()
// 	for i := 0; i < b.N; i++ {
// 		ff := []string{}
// 		results := make(chan DirScanResults, 20)
// 		wg := &sync.WaitGroup{}
// 		wg.Add(1)
// 		go func(dir string) {
// 			defer wg.Done()
// 			ScanDirRecursiveForImageFiles(dir, wg, results)
// 		}(path)
// 		go func() {
// 			wg.Wait()
// 			close(results)
// 		}()
//
// 		for r := range results {
// 			ff = append(ff, r.Path)
// 		}
// 		// fmt.Printf("\nDone::%d ", len(ff))
//
// 		// js, err := json.Marshal(ff)
// 		// if err != nil {
// 		// 	fmt.Println("")
// 		// }
// 		// fmt.Println(string(js))
// 	}
// }
