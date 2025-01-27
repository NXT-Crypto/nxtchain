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
	fmt.Println("YOUR CONNECTION STRING: " + Peer.GetConnString())

	// ~ NODE MAIN ACTIONS ~ //

}

// * PEER TO PEER * //
func createPeer(seedNode string) {
	nextutils.NewLine()
	nextutils.Debug("%s", "Creating peer...")

	handlePeerEvents := func(event string) {
		nextutils.Debug("%s", "[EVENT] "+event)
	}

	maxConnections, err := strconv.Atoi(strconv.FormatFloat(config.Fields["max_connections"].(float64), 'f', 0, 64))
	if err != nil {
		nextutils.Error("Error: max_connections is not a valid integer")
		return
	}
	default_port, err := strconv.Atoi(config.Fields["default_port"].(string))
	if err != nil {
		nextutils.Error("Error: default_port is not a valid integer")
		return
	}
	seedNodesInterface := config.Fields["seed_nodes"].([]interface{})
	seedNodes := make([]string, len(seedNodesInterface))
	for i, v := range seedNodesInterface {
		seedNodes[i] = v.(string)
	}

	if seedNode != "" {
		seedNodes = append(seedNodes, seedNode)
	}

	Peer, err := gonetic.NewPeer(handlePeerEvents, maxConnections, strconv.Itoa(default_port))
	if err != nil {
		nextutils.Error("Error creating peer: %v", err)
		return
	}
	nextutils.Debug("%s", "Peer created. Starting peer...")
	nextutils.Debug("%s", "Max connections: "+strconv.Itoa(maxConnections))
	nextutils.Debug("%s", "Peer port: "+strconv.Itoa(default_port))
	if len(seedNodes) == 0 {
		nextutils.Debug("%s", "No seed nodes available, you have to manually add them or connect.")
	} else {
		nextutils.Debug("%s", "Seed nodes: "+strings.Join(seedNodes, ", "))
	}
	go Peer.Start()
	time.Sleep(1 * time.Second)
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
