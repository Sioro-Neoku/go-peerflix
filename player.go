package main

import (
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// GenericPlayer represents most players. The stream URL will be appended to the arguments.
type GenericPlayer struct {
	Name string
	Args []string
}

// Player opens a stream URL in a video player.
type Player interface {
	Open(url string) error
}

var genericPlayers = []GenericPlayer{
	{Name: "VLC", Args: []string{"vlc"}},
	{Name: "MPV", Args: []string{"mpv"}},
	{Name: "MPlayer", Args: []string{"mplayer"}},
}

// Open the given stream in a GenericPlayer.
func (p GenericPlayer) Open(url string) error {
	command := []string{}
	if runtime.GOOS == "darwin" {
		command = []string{"open", "-a"}
	}
	command = append(command, p.Args...)
	command = append(command, url)
	return exec.Command(command[0], command[1:]...).Start()
}

// openPlayer opens a stream using the specified player and port.
func openPlayer(playerName string, port int) {
	var player Player
	playerName = strings.ToLower(playerName)
	for _, genericPlayer := range genericPlayers {
		if strings.ToLower(genericPlayer.Name) == playerName {
			player = genericPlayer
		}
	}
	if player == nil {
		log.Printf("Player '%s' is not supported. Currently supported players are: %s", playerName, joinPlayerNames())
		return
	}
	log.Printf("Playing in %s", playerName)
	if err := player.Open("http://localhost:" + strconv.Itoa(port)); err != nil {
		log.Printf("Error opening %s: %s\n", playerName, err)
	}
}

// joinPlayerNames returns a list of supported video players ready for display.
func joinPlayerNames() string {
	var names []string
	for _, player := range genericPlayers {
		names = append(names, player.Name)
	}
	return strings.Join(names, ", ")
}
