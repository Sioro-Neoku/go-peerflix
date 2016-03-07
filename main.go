package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"fmt"
)

// Exit statuses.
const (
	_ = iota
	exitNoTorrentProvided
	exitErrorInClient
)

var Version = ""
var BuildDate = ""

func main() {
	// Parse flags.
	var port int
	var seed, tcp *bool
	var player *string

	player = flag.String("player", "", "Open the stream with a video player ("+joinPlayerNames()+")")
	flag.IntVar(&port, "port", 8080, "Port to stream the video on")
	seed = flag.Bool("seed", false, "Seed after finished downloading")
	tcp = flag.Bool("tcp", true, "Allow connections via TCP")
	printVersion()
	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(exitNoTorrentProvided)
	}

	// Start up the torrent client.
	client, err := NewClient(flag.Arg(0), port, *seed, *tcp)
	if err != nil {
		log.Fatalf(err.Error())
		os.Exit(exitErrorInClient)
	}

	// Http handler.
	go func() {
		http.HandleFunc("/", client.GetFile)
		log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
	}()

	// Open selected video player
	if *player != "" {
		go func() {
			for !client.ReadyForPlayback() {
				time.Sleep(time.Second)
			}
			openPlayer(*player, port)
		}()
	}

	// Handle exit signals.
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func(interruptChannel chan os.Signal) {
		for range interruptChannel {
			log.Println("Exiting...")
			client.Close()
			os.Exit(0)
		}
	}(interruptChannel)

	// Cli render loop.
	for {
		client.Render()
		time.Sleep(time.Second)
	}
}

func printVersion() {
	fmt.Print("go-peerflix")
	if Version != "" {
		fmt.Printf(" %s", Version)
	}
	if BuildDate != "" {
		fmt.Printf(" (%s)", BuildDate)
	}
	fmt.Print("\n\n")
}