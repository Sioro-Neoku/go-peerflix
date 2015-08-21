package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/julienschmidt/httprouter"
)

var t torrent.Torrent

func main() {
	var client *torrent.Client

	flag.Bool("seed", true, "Seed after finished downloading")
	flag.Parse()
	if len(flag.Args()) == 0 {
		usage()
		os.Exit(1)
	}

	client, err := torrent.NewClient(&torrent.Config{
		DataDir:  os.TempDir(),
		NoUpload: true,
	})

	if err != nil {
		log.Fatal(err)
		os.Exit(3)
	}

	if t, err = client.AddMagnet(flag.Arg(0)); err != nil {
		log.Fatal(err)
		os.Exit(2)
	}

	go func() {
		<-t.GotInfo()
		t.DownloadAll()
	}()

	go func() {
		router := httprouter.New()
		router.GET("/", getFile)
		log.Fatal(http.ListenAndServe(":8080", router))
	}()

	for true {
		render()
		time.Sleep(time.Second)
	}
}

func render() {

	/*
		print("\033[H\033[2J")
		fmt.Println(t)
		log.Printf("t = %#v\n", t.Files())
	*/
}

func usage() {
	flag.Usage()
}

func getLargestFile() torrent.File {
	var target torrent.File
	var maxSize int64

	for _, file := range t.Files() {
		if maxSize < file.Length() {
			maxSize = file.Length()
			target = file
		}
	}

	return target
}

func getNewFileReader(f torrent.File) SeekableContent {
	reader := t.NewReader()
	reader.SetReadahead(int64(128 * 1024))
	reader.SetResponsive()
	reader.Seek(f.Offset(), os.SEEK_SET)

	return &FileEntry{
		File:   &f,
		Reader: reader,
	}
}

func getFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	target := getLargestFile()
	entry := getNewFileReader(target)
	defer entry.Close()

	http.ServeContent(w, r, target.DisplayPath(), time.Now(), entry)
}
