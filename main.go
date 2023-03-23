package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// Exit statuses.
const (
	_ = iota
	exitNoTorrentProvided
	exitErrorInClient
	exitErrSplitHostPort
)

func main() {
	// Parse flags.
	player := flag.String("player", "", "Open the stream with a video player ("+joinPlayerNames()+")")
	cfg := NewClientConfig()
	flag.IntVar(&cfg.Port, "port", 0, "Port to stream the video on")
	flag.IntVar(&cfg.TorrentPort, "torrent-port", cfg.TorrentPort, "Port to listen for incoming torrent connections")
	flag.BoolVar(&cfg.Seed, "seed", cfg.Seed, "Seed after finished downloading")
	flag.IntVar(&cfg.MaxConnections, "conn", cfg.MaxConnections, "Maximum number of connections")
	flag.BoolVar(&cfg.TCP, "tcp", cfg.TCP, "Allow connections via TCP")
	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(exitNoTorrentProvided)
	}
	cfg.TorrentPath = flag.Arg(0)

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(cfg.Port))
	if err != nil {
		log.Fatalf("net.Listen: %s", err)
	}

	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(exitErrSplitHostPort)
	}

	port, err := strconv.ParseInt(portStr, 10, 64)
	if err != nil {
		log.Fatalf("strconv.ParseInt: %s", err)
	}
	cfg.Port = int(port)

	// Start up the torrent client.
	client, err := NewClient(cfg)
	if err != nil {
		log.Fatalf(err.Error())
		os.Exit(exitErrorInClient)
	}

	// Http handler.
	go func() {
		http.HandleFunc("/", client.GetFile)
		log.Fatal(http.Serve(ln, nil))
	}()

	// Open selected video player
	if *player != "" {
		go func() {
			for !client.ReadyForPlayback() {
				time.Sleep(time.Second)
			}
			openPlayer(*player, cfg.Port)
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
