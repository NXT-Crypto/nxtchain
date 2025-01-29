package main

import (
	"flag"
	"fmt"
	"nxtchain/clitools"
	"nxtchain/configmanager"
	"nxtchain/gonetic"
	"nxtchain/nextutils"
	"strconv"
	"strings"
	"time"
)

// * GLOBAL VARS * //
var version string = "0.0.0"
var devmode bool = true

// * CONFIG * //
var config configmanager.Config

// * MAIN START * //
func main() {
	seedNode := flag.String("seednode", "", "Optional seed node IP address")
	flag.Parse()

	startup()
	createPeer(*seedNode)
}

// * MAIN * //
func start(Peer *gonetic.Peer) {
	nextutils.NewLine()
	nextutils.Debug("%s", "Beginning node main actions...")
	nextutils.Debug("%s", "Connection string: "+Peer.GetConnString())
	fmt.Println("YOUR CONNECTION STRING: " + Peer.GetConnString())

	// ~ NODE MAIN ACTIONS & LOG ~ //
	for {
		time.Sleep(1 * time.Second)
		conns := Peer.GetConnectedPeers()
		if len(conns) > 0 {
			fmt.Println(conns)
		}
	}
}

// * PEER OUTPUT HANDLER * //
func handleEvents(event string) {
	nextutils.Debug("%s", "[PEER EVENT] "+event)
}

// * PEER TO PEER * //
func createPeer(seedNode string) {
	nextutils.NewLine()
	nextutils.Debug("%s", "Creating peer...")

	maxConnections, err := strconv.Atoi(strconv.FormatFloat(config.Fields["max_connections"].(float64), 'f', 0, 64))
	if err != nil {
		nextutils.Error("Error: max_connections is not a valid integer")
		return
	}

	peerOutput := func(event string) {
		go handleEvents(event)
	}

	defaultPortStr := config.Fields["default_port"].(string)
	var default_port int
	if defaultPortStr == "" {
		default_port = 0
	} else {
		default_port, err = strconv.Atoi(defaultPortStr)
		if err != nil {
			nextutils.Error("Error: default_port is not a valid integer")
			return
		}
	}
	seedNodesInterface := config.Fields["seed_nodes"].([]interface{})
	seedNodes := make([]string, len(seedNodesInterface))
	for i, v := range seedNodesInterface {
		seedNodes[i] = v.(string)
	}

	if seedNode != "" {
		seedNodes = append(seedNodes, seedNode)
	}

	port := ""
	if default_port != 0 {
		port = strconv.Itoa(default_port)
	}
	Peer, err := gonetic.NewPeer(peerOutput, maxConnections, port)
	if err != nil {
		nextutils.Error("Error creating peer: %v", err)
		return
	}
	nextutils.Debug("%s", "Peer created. Starting peer...")
	nextutils.Debug("%s", "Max connections: "+strconv.Itoa(maxConnections))
	port = Peer.Port
	if default_port == 0 {
		nextutils.Debug("%s", "Peer port: random, see below ")
	} else {
		nextutils.Debug("%s", "Peer port: "+port)
	}
	go Peer.Start()
	time.Sleep(2 * time.Second)
	if len(seedNodes) == 0 {
		nextutils.Debug("%s", "No seed nodes available, you have to manually add them or connect.")
	} else {
		nextutils.Debug("%s", "Seed nodes: "+strings.Join(seedNodes, ", "))
		nextutils.Debug("%s", "Connecting to seed nodes...")
		for _, seedNode := range seedNodes {
			go Peer.Connect(seedNode)
		}

	}
	go clitools.UpdateCmdTitle(Peer)
	start(Peer)
}

// * STARTUP * //
func startup() {
	nextutils.InitDebugger(false)
	nextutils.NewLine()
	nextutils.Debug("Starting node...")
	nextutils.Debug("%s", "Version: "+version)
	nextutils.Debug("%s", "Developer Mode: "+strconv.FormatBool(devmode))
	nextutils.NewLine()
	nextutils.Debug("%s", "Checking config file...")
	configmanager.InitConfig()
	nextutils.Debug("%s", "Config file: "+configmanager.GetConfigPath())

	nextutils.Debug("%s", "Applying config...")
	var err error
	config, err = configmanager.LoadConfig()
	if err != nil {
		nextutils.Error("Error loading config: %v", err)
		return
	}
	if err := configmanager.SetItem("block_dir", "blocks", &config, true); err != nil {
		nextutils.Error("Error setting block_dir: %v", err)
		return
	}
	if err := configmanager.SetItem("default_port", "5012", &config, true); err != nil {
		nextutils.Error("Error setting default_port: %v", err)
		return
	}
	nextutils.Debug("%s", "Config applied.")
	for key, value := range config.Fields {
		nextutils.Debug("- %s = %v", key, value)
	}

	nextutils.PrintLogo("V "+version+" - (c) 2025 NXTCHAIN. All rights reserved.\n-> NODE APPLICATION", devmode)
}
