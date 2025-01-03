package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

var (
	singlePage = flag.Bool("sp", false, "single page: redirect ingest to a single file when file not found")
	port       = flag.String("p", "8080", "port to serve on")
	directory  = flag.String("d", ".", "the directory of static file to host")
)

type Handler struct{}

func main() {
	flag.Parse()

	handler := Handler{}

	log.Printf("Serving %s on http port: %s\n", *directory, *port)
	log.Fatal(http.ListenAndServe(":"+*port, handler))
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	folderPath, _ := filepath.Abs(*directory)
	folderPath += "/"
	uriPathRaw := r.URL.Path
	if uriPathRaw == "/" {
		uriPathRaw = "/index.html"
	}
	uriPath := fmt.Sprintf("%s/%s", folderPath, path.Clean(uriPathRaw))

	file, stat, errFileStats := FileStats(uriPath)
	defer file.Close()
	if errors.Is(errFileStats, os.ErrNotExist) {
		if *singlePage {
			fileIndex, statIndex, errIndex := FileStats("index.html")
			if errIndex != nil {
				r.URL.Path = "/"
				http.FileServer(http.Dir(folderPath)).ServeHTTP(w, r)
				return
			}
			defer fileIndex.Close()

			http.ServeContent(w, r, statIndex.Name(), statIndex.ModTime(), fileIndex)
		} else {

			http.Error(w, errFileStats.Error(), http.StatusInternalServerError)
		}
		return

	} else if errFileStats != nil {

		http.Error(w, errFileStats.Error(), http.StatusInternalServerError)
		return
	}

	if stat.IsDir() {
		http.FileServer(http.Dir(folderPath)).ServeHTTP(w, r)
		return
	}

	http.ServeContent(w, r, uriPath, stat.ModTime(), file)
}

func FileStats(filename string) (*os.File, os.FileInfo, error) {
	file, errOpen := os.Open(filename)
	if errOpen != nil {
		return nil, nil, errOpen
	}

	stat, errStat := file.Stat()
	if errStat != nil {
		return nil, nil, errStat
	}

	return file, stat, nil
}
