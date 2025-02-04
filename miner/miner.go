package main

import (
	"flag"
	"fmt"
	"nxtchain/clitools"
	"nxtchain/configmanager"
	"nxtchain/gonetic"
	"nxtchain/nextutils"
	"nxtchain/nxtblock"
	"nxtchain/nxtutxodb"
	"strconv"
	"strings"
	"time"
)

// TODO //

// 1. Blocks & UTXO Datenbank synchronisieren

// * GLOBAL VARS * //
var version string = "0.0.0"
var devmode bool = true
var blockHeightCounts = make(map[int]int)
var utxodbs = make(map[string]map[string]nxtutxodb.UTXO)
var totalResponses int
var totalUDBresponses int
var blockdir string = "blocks"
var remainingBlockHeights int
var remainingDBs int

// * CONFIG * //
var config configmanager.Config

// * MAIN START * //
func main() {
	seedNode := flag.String("seednode", "", "Optional seed node IP address")
	debug := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	startup(debug)
	createPeer(*seedNode)
}

// * MAIN * //
func start(peer *gonetic.Peer) {
	nextutils.NewLine()
	nextutils.Debug("%s", "Beginning miner main actions...")
	nextutils.Debug("%s", "Connection string: "+peer.GetConnString())
	fmt.Println("+- YOUR CONNECTION STRING: " + peer.GetConnString())
	nextutils.NewLine()
	nextutils.Debug("%s", "Waiting for peers to sync...")
	fmt.Println("+- WAITING FOR PEERS TO SYNC -")
	for {
		if len(peer.GetConnectedPeers()) > 0 {
			nextutils.Debug("%s", "Starting syncronization...")
			nextutils.Debug("%s", "Syncing Blockchain...")
			syncBlockchain(peer)
			syncUTXODB(peer)
			nextutils.Debug("%s", "Syncronization complete.")
			fmt.Println("+- SYNC COMPLETE -")
			break
		}
		time.Sleep(1 * time.Second)

	}

	// ~ NODE MAIN ACTIONS & LOG ~ //
	for {
		var input string
		msg, err := fmt.Scanln(&input)
		if err != nil {
			nextutils.Error("Error: %v", msg)
		}
		if strings.HasPrefix(input, "$") {
			if strings.HasPrefix(input, "$exit") {
				nextutils.Debug("%s", "Exiting miner...")
				peer.Stop()
				nextutils.Debug("%s", "Miner stopped.")
				nextutils.NewLine()
				nextutils.Debug("%s", "Goodbye!")
				break
			} else if strings.HasPrefix(input, "$connections") {
				connected := peer.GetConnectedPeers()
				fmt.Println("+- CONNECTED PEERS -")
				for _, conn := range connected {
					fmt.Println("+- " + conn)
				}
			} else if strings.HasPrefix(input, "$blockheight") {
				blockh := nxtblock.GetLocalBlockHeight(blockdir)
				fmt.Println("+- BLOCK HEIGHT -")
				fmt.Println("+- " + strconv.Itoa(blockh))
			} else if strings.HasPrefix(input, "$sync") {
				nextutils.NewLine()
				nextutils.Debug("%s", "Starting syncronization...")
				nextutils.Debug("%s", "Syncing Blockchain...")
				syncBlockchain(peer)
				syncUTXODB(peer)
				nextutils.Debug("%s", "Syncronization complete.")
				fmt.Println("+- SYNC COMPLETE -")
			} else if strings.HasPrefix(input, "$restart") {
				start(peer)
			}
		} else {
			peer.Broadcast(input)
		}
	}
}

// * PEER OUTPUT HANDLER * //
func handleEvents(event string, peer *gonetic.Peer) {
	nextutils.Debug("%s", "[PEER EVENT] "+event)

	// ~ PEER EVENTS ~ //
	parts := strings.SplitN(event, "_", 2)
	nextutils.Debug("%s", "Event parts: "+strings.Join(parts, ", "))
	if len(parts) < 2 {
		nextutils.Error("%s", "Invalid event format: "+event)
		return
	}
	event_header := parts[0]
	event_body := parts[1]

	switch event_header {
	case "RGET": // * GET - ANFRAGEN * //
		nextutils.Debug("%s", "[GET] "+event_body)

		if strings.HasPrefix(event_body, "BLOCKHEIGHT_") {
			parts := strings.Split(event_body, "_")
			requester := ""
			nextutils.Debug("%s", "Sending block height to: "+requester)
			if len(parts) > 1 {
				requester = parts[1]
			}
			blockHeight := nxtblock.GetLocalBlockHeight(blockdir)
			peer.SendToPeer(requester, "RESPONSE_BLOCKHEIGHT_"+strconv.Itoa(blockHeight))
			nextutils.Debug("%s", "[+] Sent block height to: "+requester+" ("+strconv.Itoa(blockHeight)+")")
		} else if strings.HasPrefix(event_body, "BLOCK_") {
			parts := strings.Split(event_body, "_")
			heightStr := parts[1]
			requester := parts[2]
			nextutils.Debug("%s", "Sending block to: "+requester)
			height, err := strconv.Atoi(heightStr)
			if err != nil {
				nextutils.Error("Invalid block height: %s", heightStr)
				return
			}
			block, err := nxtblock.GetBlockByHeight(height, blockdir)
			if err != nil {
				nextutils.Error("Error: %v", err)
				return
			}
			blockStr, err := nxtblock.PrepareBlockSender(block)
			if err != nil {
				nextutils.Error("Error: %v", err)
				return
			}
			peer.SendToPeer(requester, "RESPONSE_BLOCK_"+blockStr)
			nextutils.Debug("%s", "[+] Sent block to: "+requester)

		} else if strings.HasPrefix(event_body, "UTXODB_") {
			parts := strings.Split(event_body, "_")
			requester := parts[1]
			nextutils.Debug("%s", "Sending UTXO DB to: "+requester)
			utxoDB := nxtutxodb.GetUTXODatabase()
			if utxoDB == nil {
				utxoDB = make(map[string]nxtutxodb.UTXO)
			}
			utxoDBStr, err := nxtblock.PrepareUTXOSender(utxoDB)
			if err != nil {
				nextutils.Error("Error: %v", err)
				return
			}
			peer.SendToPeer(requester, "RESPONSE_UTXODB_"+utxoDBStr)
			nextutils.Debug("%s", "[+] Sent UTXO DB to: "+requester+" ("+strconv.Itoa(len(utxoDB))+" entries)")
		}

	case "RESPONSE": // * RESPONSE - ANTWORTEN AUF DEINE ANFRAGEN * //
		nextutils.Debug("%s", "[RESPONSE] "+event_body)

		if strings.HasPrefix(event_body, "BLOCKHEIGHT_") {
			heightStr := strings.TrimPrefix(event_body, "BLOCKHEIGHT_")
			heightStr = strings.TrimSpace(heightStr)
			blockHeight, err := strconv.Atoi(heightStr)
			if err != nil {
				nextutils.Error("Error converting block height: %v", err)
				return
			}
			if blockHeight < 0 {
				nextutils.Error("Invalid block height: %d", blockHeight)
				return
			}

			blockHeightCounts[blockHeight]++
			totalResponses++

			nextutils.Debug("Block height: %d (%d/%d responses)", blockHeight, totalResponses, remainingBlockHeights)

			if totalResponses >= remainingBlockHeights { // * ALL RESPONSES FOR A VALID * //
				selectedHeight := getMostFrequentBlockHeight()
				nextutils.Debug("Selected block height for sync: %d", selectedHeight)
				startBlockchainSync(selectedHeight, peer)
			}

		} else if strings.HasPrefix(event_body, "UTXODB_") {
			utxoDBStr := strings.TrimPrefix(event_body, "UTXODB_")
			utxoDB, err := nxtblock.GetUTXOSender(utxoDBStr)
			if err != nil {
				nextutils.Error("Error: %v", err)
				return
			}

			utxodbs[peer.GetConnString()] = utxoDB
			totalUDBresponses++

			nextutils.Debug("UTXO DB response from: %s (%d/%d responses)", peer.GetConnString(), totalUDBresponses, remainingDBs)

			if totalUDBresponses >= remainingDBs { // * ALL RESPONSES FOR A VALID * //
				selectedDB := getMostFrequentUTXODB()
				nextutils.Debug("Selected UTXO DB for sync: %v", selectedDB)
				nxtutxodb.SetUTXODatabase(selectedDB)
			}
		}
	case "NEW": // * NEW - NEUE OBJEKTE * //
		parts := strings.SplitN(event_body, "_", 2)
		if len(parts) < 2 {
			nextutils.Error("%s", "Invalid event body format: "+event_body)
			return
		}
		newType := parts[0]
		newObject := parts[1]
		switch newType {
		case "TRANSACTION":
			newTransaction, err := nxtblock.GetTransactionSender(newObject)
			if err != nil {
				nextutils.Error("Error: %v", err)
				return
			}

			// * VALIDATE TRANSACTION * //
			//! JUST FOR TESTING WITHOUT SYNCRO OF UTXO DB nxtutxodb.AddUTXO("txid1", 1, 100000000000000, "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", 0, false)
			nextutils.Debug("%s", "Validating transaction (ID: "+newTransaction.ID+")...")
			valid, err := nxtblock.ValidatorValidateTransaction(newTransaction)
			if err != nil {
				nextutils.Error("%s", "Error: Transaction (ID: "+newTransaction.ID+") is not valid")
				nextutils.Error("Error: %v", err)
				nextutils.Error(fmt.Sprintf("UTXO Database (formatted): %+v", nxtutxodb.GetUTXODatabase()))
				return
			}
			if !valid {
				nextutils.Error("%s", "Error: Transaction (ID: "+newTransaction.ID+") is not valid")
				nextutils.Error("%s", "UTXO Database (formatted): %+v", nxtutxodb.GetUTXODatabase())
				return
			}
			nextutils.Debug("%s", "Transaction (ID: "+newTransaction.ID+") is valid.")
			nxtblock.AddTransactionToPool(newTransaction)
			fmt.Println("[+] Added transaction: #" + newTransaction.ID + " to the mempool")
			nextutils.Debug("%s", "Mempool size: "+strconv.Itoa(len(nxtblock.GetAllTransactionsFromPool())))

		default:
			nextutils.Debug("%s", "Unknown new object: "+newObject)
		}

	default:
		nextutils.Debug("%s", "Unknown event: "+event)
	}
}

// * SYNC BLOCKCHAIN * //

func syncBlockchain(peer *gonetic.Peer) {
	nextutils.Debug("%s", "starting syncronization of Blockchain")
	remainingBlockHeights = len(peer.GetConnectedPeers())
	// ? Broadcasten: CURRENT_BLOCKHEIGHT -> Die meisten blockheights gewinnen
	// ? Ausrechnen welche blöcke dir fehlen (keine blöcke=0) (MY_BLOCKHEIGHT - CURRENT_BLOCKHEIGHT)
	// ? Broadcasten: GET_BLOCK_x (x ist blockheight)
	// * DIE anderen peers schicken die blöcke zurück privat an dich zurück

	peer.Broadcast("RGET_BLOCKHEIGHT_" + peer.GetConnString())

}
func startBlockchainSync(selectedHeight int, peer *gonetic.Peer) {
	nextutils.Debug("Starting blockchain sync for block height: %d", selectedHeight)

	localHeight := nxtblock.GetLocalBlockHeight(blockdir)
	remainingBlockHeights = localHeight - selectedHeight
	total := float64(remainingBlockHeights)

	for i := selectedHeight; i < localHeight; i++ {
		progress := float64(i-selectedHeight) / total * 100
		fmt.Printf("\rSynchronizing blocks: %.1f%% (%d/%d)", progress, i-selectedHeight, remainingBlockHeights)
		peer.Broadcast("RGET_BLOCK_" + strconv.Itoa(i) + "_" + peer.GetConnString())
	}
	fmt.Printf("\rSynchronizing blocks: 100.0%% (%d/%d)\n", remainingBlockHeights, remainingBlockHeights)
	nextutils.Debug("Block synchronization requests completed")
}
func getMostFrequentBlockHeight() int {
	var maxHeight, maxCount int
	for height, count := range blockHeightCounts {
		if count > maxCount {
			maxHeight = height
			maxCount = count
		}
	}
	return maxHeight
}

// * SYNC UTXO DATABASE * //

func syncUTXODB(peer *gonetic.Peer) {
	nextutils.Debug("%s", "Syncing UTXO DB...")
	// ? Broadcasten und alle UTXO datenbanken hashen, der meiste hash gewinnt und wird dann gesetzt
	remainingDBs = len(peer.GetConnectedPeers())
	peer.Broadcast("RGET_UTXODB_" + peer.GetConnString())

}
func compareMaps(m1, m2 map[string]nxtutxodb.UTXO) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v1 := range m1 {
		v2, ok := m2[k]
		if !ok || v1 != v2 {
			return false
		}
	}
	return true
}

func getMostFrequentUTXODB() map[string]nxtutxodb.UTXO {
	var maxDB map[string]nxtutxodb.UTXO
	var maxCount int
	for _, utxoDB := range utxodbs {
		count := 0
		for _, db := range utxodbs {
			if compareMaps(db, utxoDB) {
				count++
			}
		}
		if count > maxCount {
			maxDB = utxoDB
			maxCount = count
		}
	}
	return maxDB
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

	var peer *gonetic.Peer
	peerOutput := func(event string) {
		go handleEvents(event, peer)
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
	peer, err = gonetic.NewPeer(peerOutput, maxConnections, port)
	if err != nil {
		nextutils.Error("Error creating peer: %v", err)
		return
	}
	nextutils.Debug("%s", "Peer created. Starting peer...")
	nextutils.Debug("%s", "Max connections: "+strconv.Itoa(maxConnections))
	port = peer.Port
	if default_port == 0 {
		nextutils.Debug("%s", "Peer port: random, see below ")
	} else {
		nextutils.Debug("%s", "Peer port: "+port)
	}
	go peer.Start()
	time.Sleep(2 * time.Second)
	if len(seedNodes) == 0 {
		nextutils.Debug("%s", "No seed nodes available, you have to manually add them or connect.")
	} else {
		nextutils.Debug("%s", "Seed nodes: "+strings.Join(seedNodes, ", "))
		nextutils.Debug("%s", "Connecting to seed nodes...")
		for _, seedNode := range seedNodes {
			go peer.Connect(seedNode)
		}
	}
	go clitools.UpdateCmdTitle(peer)
	for peer == nil {
		time.Sleep(100 * time.Millisecond)
	}
	start(peer)
}

// * STARTUP * //
func startup(debug *bool) {
	nextutils.InitDebugger(*debug)
	nextutils.NewLine()
	nextutils.Debug("Starting miner...")
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

	if config.Fields["block_dir"] != nil {
		blockdir = config.Fields["block_dir"].(string)
	}

	nextutils.PrintLogo("V "+version+" - (c) 2025 NXTCHAIN. All rights reserved.\n-> NODE APPLICATION", devmode)
}
