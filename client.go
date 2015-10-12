package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/dustin/go-humanize"
)

type Client struct {
	Client  *torrent.Client
	Torrent torrent.Torrent
}

func NewClient(data string) (client Client, err error) {
	var t torrent.Torrent

	// Create client.
	c, err := torrent.NewClient(&torrent.Config{
		DataDir:  os.TempDir(),
		NoUpload: !(*seed),
	})

	client.Client = c

	// Add magnet url.
	t, err = c.AddMagnet(data)

	client.Torrent = t

	if err == nil {
		go func() {
			<-t.GotInfo()
			t.DownloadAll()
		}()
	}

	return
}

func (c Client) Render() {
	t := c.Torrent

	var currentProgress = c.Torrent.BytesCompleted()
	speed := humanize.Bytes(uint64(currentProgress-progress)) + "/s"
	progress = currentProgress

	percentage := float64(t.BytesCompleted()) / float64(t.Length()) * 100
	complete := humanize.Bytes(uint64(t.BytesCompleted()))
	size := humanize.Bytes(uint64(t.Length()))
	connections := len(t.Conns)

	print(clearScreen)
	fmt.Println(t.Name())
	fmt.Println("=============================================================")
	if t.BytesCompleted() > 0 {
		fmt.Printf("Progress: \t%s / %s  %.2f%%\n", complete, size, percentage)
	}
	if t.BytesCompleted() < t.Length() {
		fmt.Printf("Download speed: %s\n", speed)
	}
	fmt.Printf("Connections: \t%d\n", connections)
}

func (c Client) getLargestFile() torrent.File {
	var target torrent.File
	var maxSize int64

	for _, file := range c.Torrent.Files() {
		if maxSize < file.Length() {
			maxSize = file.Length()
			target = file
		}
	}

	return target
}

func (c Client) ReadyForPlayback() bool {
	percentage := float64(c.Torrent.BytesCompleted()) / float64(c.Torrent.Length())

	return percentage > 0.05
}

func (c Client) GetFile(w http.ResponseWriter, r *http.Request) {
	target := c.getLargestFile()
	entry, err := NewFileReader(c, target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() {
		if err := entry.Close(); err != nil {
			log.Printf("Error closing file reader: %s\n", err)
		}
	}()

	w.Header().Set("Content-Disposition", "attachment; filename=\""+c.Torrent.Name()+"\"")
	http.ServeContent(w, r, target.DisplayPath(), time.Now(), entry)
}
