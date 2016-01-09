package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/iplist"
	"github.com/dustin/go-humanize"
)

const clearScreen = "\033[H\033[2J"

const torrentBlockListURL = "http://john.bitsurge.net/public/biglist.p2p.gz"

var isHTTP = regexp.MustCompile(`^https?:\/\/`)

// ClientError formats errors coming from the client.
type ClientError struct {
	Type   string
	Origin error
}

func (clientError ClientError) Error() string {
	return fmt.Sprintf("Error %s: %s\n", clientError.Type, clientError.Origin)
}

// Client manages the torrent downloading.
type Client struct {
	Client   *torrent.Client
	Torrent  torrent.Torrent
	Progress int64
	Port     int
}

// NewClient creates a new torrent client based on a magnet or a torrent file.
// If the torrent file is on http, we try downloading it.
func NewClient(torrentPath string, port int, seed bool) (client Client, err error) {
	var t torrent.Torrent
	var c *torrent.Client

	client.Port = port

	// Create client.
	c, err = torrent.NewClient(&torrent.Config{
		DataDir:  os.TempDir(),
		NoUpload: !seed,
		Seed:     seed,
	})

	if err != nil {
		return client, ClientError{Type: "creating torrent client", Origin: err}
	}

	client.Client = c

	// Add torrent.

	// Add as magnet url.
	if strings.HasPrefix(torrentPath, "magnet:") {
		if t, err = c.AddMagnet(torrentPath); err != nil {
			return client, ClientError{Type: "adding torrent", Origin: err}
		}
	} else {
		// Otherwise add as a torrent file.

		// If it's online, we try downloading the file.
		if isHTTP.MatchString(torrentPath) {
			if torrentPath, err = downloadFile(torrentPath); err != nil {
				return client, ClientError{Type: "downloading torrent file", Origin: err}
			}
		}

		// Check if the file exists.
		if _, err = os.Stat(torrentPath); err != nil {
			return client, ClientError{Type: "file not found", Origin: err}
		}

		if t, err = c.AddTorrentFromFile(torrentPath); err != nil {
			return client, ClientError{Type: "adding torrent to the client", Origin: err}
		}
	}

	client.Torrent = t

	go func() {
		<-t.GotInfo()
		t.DownloadAll()

		// Prioritize first 5% of the file.
		client.getLargestFile().PrioritizeRegion(0, int64(t.NumPieces()/100*5))
	}()

	go client.addBlocklist()

	return
}

// Download and add the blocklist.
func (c *Client) addBlocklist() {
	if c.Client.IPBlockList() != nil {
		log.Printf("Found blocklist")
		return
	}

	var err error
	blocklistPath := c.Client.ConfigDir() + "/blocklist"

	// Download blocklist.
	log.Printf("Downloading blocklist")
	fileName, err := downloadFile(torrentBlockListURL)
	if err != nil {
		log.Printf("Error downloading blocklist: %s\n", err)
		return
	}

	// Ungzip file.
	in, err := os.Open(fileName)
	if err != nil {
		log.Printf("Error extracting blocklist: %s\n", err)
		return
	}
	defer func() {
		if err := in.Close(); err != nil {
			log.Printf("Error closing the blocklist gzip file: %s", err)
		}
	}()
	reader, err := gzip.NewReader(in)

	// Write to {configdir}/blocklist
	out, err := os.Create(blocklistPath)
	if err != nil {
		log.Printf("Error writing blocklist: %s\n", err)
		return
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Printf("Error writing the blocklist file: %s", err)
		}
	}()
	_, err = io.Copy(out, reader)
	if err != nil {
		log.Printf("Error writing the blocklist file: %s", err)
		return
	}

	// Add to the blocklist.
	blocklistReader, err := os.Open(blocklistPath)
	blocklist, err := iplist.NewFromReader(blocklistReader)
	if err != nil {
		log.Printf("Error reading blocklist: %s", err)
		return
	}
	log.Printf("Setting blocklist.\nFound %d ranges\n", blocklist.NumRanges())
	c.Client.SetIPBlockList(blocklist)
}

// Close cleans up the connections.
func (c *Client) Close() {
	c.Torrent.Drop()
	c.Client.Close()
}

// Render outputs the command line interface for the client.
func (c *Client) Render() {
	t := c.Torrent

	if t.Info() == nil {
		return
	}

	var currentProgress = t.BytesCompleted()
	speed := humanize.Bytes(uint64(currentProgress-c.Progress)) + "/s"
	c.Progress = currentProgress

	complete := humanize.Bytes(uint64(currentProgress))
	size := humanize.Bytes(uint64(t.Info().TotalLength()))

	print(clearScreen)
	fmt.Println(t.Info().Name)
	fmt.Println("=============================================================")
	if c.ReadyForPlayback() {
		fmt.Printf("Stream: \thttp://localhost:%d\n", c.Port)
	}

	if currentProgress > 0 {
		fmt.Printf("Progress: \t%s / %s  %.2f%%\n", complete, size, c.percentage())
	}
	if currentProgress < t.Info().TotalLength() {
		fmt.Printf("Download speed: %s\n", speed)
	}
	fmt.Printf("Connections: \t%d\n", len(t.Peers))
	//fmt.Printf("%s\n", c.RenderPieces())
}

func (c Client) getLargestFile() *torrent.File {
	var target torrent.File
	var maxSize int64

	for _, file := range c.Torrent.Files() {
		if maxSize < file.Length() {
			maxSize = file.Length()
			target = file
		}
	}

	return &target
}

/*
func (c Client) RenderPieces() (output string) {
	pieces := c.Torrent.PieceStateRuns()
	for i := range pieces {
		piece := pieces[i]

		if piece.Priority == torrent.PiecePriorityReadahead {
			output += "!"
		}

		if piece.Partial {
			output += "P"
		} else if piece.Checking {
			output += "c"
		} else if piece.Complete {
			output += "d"
		} else {
			output += "_"
		}
	}

	return
}
*/

// ReadyForPlayback checks if the torrent is ready for playback or not.
// We wait until 5% of the torrent to start playing.
func (c Client) ReadyForPlayback() bool {
	return c.percentage() > 5
}

// GetFile is an http handler to serve the biggest file managed by the client.
func (c Client) GetFile(w http.ResponseWriter, r *http.Request) {
	target := c.getLargestFile()
	entry, err := NewFileReader(target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() {
		if err := entry.Close(); err != nil {
			log.Printf("Error closing file reader: %s\n", err)
		}
	}()

	w.Header().Set("Content-Disposition", "attachment; filename=\""+c.Torrent.Info().Name+"\"")
	http.ServeContent(w, r, target.DisplayPath(), time.Now(), entry)
}

func (c Client) percentage() float64 {
	info := c.Torrent.Info()

	if info == nil {
		return 0
	}

	return float64(c.Torrent.BytesCompleted()) / float64(info.TotalLength()) * 100
}

func downloadFile(URL string) (fileName string, err error) {
	var file *os.File
	if file, err = ioutil.TempFile(os.TempDir(), "torrent-imageviewer"); err != nil {
		return
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing torrent file: %s", err)
		}
	}()

	response, err := http.Get(URL)
	if err != nil {
		return
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Printf("Error closing torrent file: %s", err)
		}
	}()

	_, err = io.Copy(file, response.Body)

	return file.Name(), err
}
