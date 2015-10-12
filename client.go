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

const clearScreen = "\033[H\033[2J"

// ClientError formats errors coming from the client.
type ClientError struct {
	Type   string
	Origin error
}

func (clientError ClientError) Error() string {
	return fmt.Sprintf("Error %s: %s", clientError.Type, clientError.Origin)
}

// Client manages the torrent downloading.
type Client struct {
	Client   *torrent.Client
	Torrent  torrent.Torrent
	Progress int64
}

// NewClient creates a new torrent client based on a magnet url.
func NewClient(data string) (client Client, err error) {
	var t torrent.Torrent
	var c *torrent.Client

	// Create client.
	c, err = torrent.NewClient(&torrent.Config{
		DataDir:  os.TempDir(),
		NoUpload: !(*seed),
	})

	if err != nil {
		return client, ClientError{Type: "creating torrent client", Origin: err}
	}

	client.Client = c

	// Add magnet url.
	if t, err = c.AddMagnet(data); err != nil {
		return client, ClientError{Type: "adding torrent", Origin: err}
	}

	client.Torrent = t

	go func() {
		<-t.GotInfo()
		t.DownloadAll()
	}()

	return
}

// Render outputs the command line interface for the client.
func (c *Client) Render() {
	t := c.Torrent

	var currentProgress = c.Torrent.BytesCompleted()
	speed := humanize.Bytes(uint64(currentProgress-c.Progress)) + "/s"
	c.Progress = currentProgress

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

// ReadyForPlayback checks if the torrent is ready for playback or not.
// we wait until 5% of the torrent to start playing.
func (c Client) ReadyForPlayback() bool {
	percentage := float64(c.Torrent.BytesCompleted()) / float64(c.Torrent.Length())

	return percentage > 0.05
}

// GetFile is an http handler to serve the biggest file managed by the client.
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
