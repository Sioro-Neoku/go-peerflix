package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

var seed *bool
var vlc *bool

// Exit statuses.
const (
	_ = iota
	exitNoTorrentProvided
	exitErrorInClient
)

func main() {
	var port int

	// Set up flags.
	vlc = flag.Bool("vlc", false, "Open vlc to play the file")
	flag.IntVar(&port, "port", 8080, "Port to stream the video on")
	seed = flag.Bool("seed", true, "Seed after finished downloading")
	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(exitNoTorrentProvided)
	}

	// Start up the torrent client.
	client, err := NewClient(flag.Arg(0), port)
	if err != nil {
		log.Fatalf(err.Error())
		os.Exit(exitErrorInClient)
	}

	// Http handler.
	go func() {
		http.HandleFunc("/", client.GetFile)
		log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
	}()

	// Open vlc to play.
	if *vlc {
		go func() {
			for !client.ReadyForPlayback() {
				time.Sleep(time.Second)
			}
			log.Printf("Playing in vlc")

			// @todo decide command to run based on os.
			if err := exec.Command("open", "-a", "vlc", "http://localhost:"+strconv.Itoa(port)).Start(); err != nil {
				log.Printf("Error opening vlc: %s\n", err)
			}
		}()
	}

	// Cli render loop.
	for {
		client.Render()
		time.Sleep(time.Second)
	}
}
