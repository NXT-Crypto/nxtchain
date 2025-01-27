package clitools

import (
	"fmt"
	"nxtchain/gonetic"
	"os"
	"os/exec"
	"time"
)

func ClearScreen() {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func changeCmdTitle(title string) {
	go func() {
		fmt.Printf("\x1b]2;%s\x07", title)
	}()
}

func UpdateCmdTitle(peer *gonetic.Peer) {
	for {
		connectedPeers := peer.GetConnectedPeers()
		title := fmt.Sprintf("Connected Peers: %d", len(connectedPeers))
		changeCmdTitle(title)
		time.Sleep(1 * time.Second)
	}
}
