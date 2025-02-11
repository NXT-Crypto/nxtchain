package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
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
var timeTargetMin float64 = 10

// * CONFIG * //
var config configmanager.Config
var ruleset nxtblock.RuleSet

// * MAIN START * //
func main() {
	seedNode := flag.String("seednode", "", "Optional seed node IP address")
	debug := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	startup(debug)
	createPeer(*seedNode)
}

// * MAIN * //
func start(Peer *gonetic.Peer) {
	if !devmode {
		clitools.ClearScreen()
	}

	//! TEST
	nxtutxodb.AddUTXO("1", 0, 100000000000000, "a0352577bbb6e354f672df9ea093f8b8146b3e9e", 1, false)
	//! -----

	nextutils.NewLine()
	nextutils.Debug("%s", "Beginning node main actions...")
	nextutils.Debug("%s", "Connection string: "+Peer.GetConnString())
	fmt.Println("YOUR CONNECTION STRING: " + Peer.GetConnString())

	// ~ NODE MAIN ACTIONS & LOG ~ //
	for {
		var input string
		msg, err := fmt.Scanln(&input)
		if err != nil {
			nextutils.Error("Error: %v", msg)
		}
		if strings.HasPrefix(input, "$") {
			if strings.HasPrefix(input, "$exit") {
				nextutils.Debug("%s", "Exiting node...")
				Peer.Stop()
				nextutils.Debug("%s", "Miner stopped.")
				nextutils.NewLine()
				nextutils.Debug("%s", "Goodbye!")
				break
			} else if strings.HasPrefix(input, "$connections") {
				connected := Peer.GetConnectedPeers()
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
				syncBlockchain(Peer)
				syncUTXODB(Peer)
				nextutils.Debug("%s", "Syncronization complete.")
				fmt.Println("+- SYNC COMPLETE -")
			} else if strings.HasPrefix(input, "$restart") {
				start(Peer)
			}
		} else {
			Peer.Broadcast(input)
		}
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

// * DIFFICULTY ADJUSTER * //

func adjustDifficulty() {
	nextutils.Debug("%s", "Adjusting difficulty...")
	// ? Calculate average time between blocks
	// ? Compare to target time
	// ? Adjust difficulty

	lblocks, err := nxtblock.GetLatestBlocks(blockdir, 20)
	if err != nil {
		nextutils.Error("Error getting latest blocks: %v", err)
		return
	}
	blockPtrs := make([]*nxtblock.Block, len(lblocks))
	for i := range lblocks {
		blockPtrs[i] = &lblocks[i]
	}

	avgTime := nxtblock.CheckBlockTimestampForDifficulty(blockPtrs)
	if avgTime == 0 {
		nextutils.Error("%s", "Not enough blocks to calculate difficulty")
		return
	}

	nextutils.Debug("Average time between blocks: %f", avgTime)
	nextutils.Debug("Target time: %f", timeTargetMin)
	avgTime = math.Round(avgTime)
	valid := avgTime < timeTargetMin
	nextutils.Debug("Need to change difficulty? %t", valid)
	direction := "increase"
	if avgTime > timeTargetMin+1 {
		direction = "decrease"
		ruleset.Difficulty--
		configmanager.SetItem("ruleset", ruleset, &config, true)
	} else if math.Abs(avgTime-timeTargetMin) <= 1 {
		direction = "do nothing"
	}
	if direction == "increase" {
		ruleset.Difficulty++
		configmanager.SetItem("ruleset", ruleset, &config, true)
	}
	nextutils.Debug("Difficulty should %s", direction)
	time.Sleep(5 * time.Minute)
}

// * PEER OUTPUT HANDLER * //
func handleEvents(event string, peer *gonetic.Peer) {
	adjustDifficulty()
	nextutils.Debug("%s", "[PEER EVENT] "+event)

	// ? INPUT REQUESTS
	// ? BLOCK REQUESTS & VALIDATION
	// ? TRANSACTION REQUESTS

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
	case "RGET":
		nextutils.Debug("%s", "[GET] "+event_body)
		if strings.HasPrefix(event_body, "INPUTS_") {
			parts := strings.Split(event_body, "_")
			if len(parts) >= 3 {
				walletAddr := parts[1]
				requesterConn := parts[2]
				nextutils.Debug("%s", "Sending inputs to: "+requesterConn+" for wallet: "+walletAddr)

				inputs := nxtutxodb.GetUTXOByWalletAddr(walletAddr)
				inputsJson, err := json.Marshal(inputs)
				if err != nil {
					nextutils.Error("Error marshaling inputs: %v", err)
					return
				}
				nextutils.Debug("%s", "[INPUTS] "+string(inputsJson))

				peer.Broadcast("RESPONSE_INPUTS_" + string(inputsJson))
				nextutils.Debug("%s", "[+] Sent inputs to: "+requesterConn+" for wallet: "+walletAddr)
			} else {
				nextutils.Error("%s", "Invalid INPUTS request format")
			}
		} else if strings.HasPrefix(event_body, "BALANCE_") {
			parts := strings.Split(event_body, "_")
			if len(parts) >= 3 {
				walletAddr := parts[1]
				requesterConn := parts[2]
				nextutils.Debug("%s", "Sending balance to: "+requesterConn+" for wallet: "+walletAddr)

				amount := nxtutxodb.GetUTXOByWalletAddr(walletAddr)
				amountJson, err := json.Marshal(amount)

				if err != nil {
					nextutils.Error("Error marshaling balance: %v", err)
					return
				}

				nextutils.Debug("%s", "[BALANCE] "+string(amountJson))
				peer.Broadcast("RESPONSE_BALANCE_" + string(amountJson) + "_" + walletAddr)
			}
		} else if strings.HasPrefix(event_body, "TRANSACTIONS_") {
			parts := strings.Split(event_body, "_")
			if len(parts) >= 3 {
				walletAddr := parts[1]
				requesterConn := parts[2]
				nextutils.Debug("%s", "Sending transactions to: "+requesterConn+" for wallet: "+walletAddr)

				transactions := nxtblock.GetAllTransactionsFromBlocks(blockdir, walletAddr)

				transactionsJson, err := json.Marshal(transactions)
				if err != nil {
					nextutils.Error("Error marshaling transactions: %v", err)
					return
				}

				nextutils.Debug("%s", "[TRANSACTIONS] "+string(transactionsJson))
				peer.Broadcast("RESPONSE_TRANSACTIONS_" + string(transactionsJson) + "_" + walletAddr)

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
			nextutils.Debug("%s", "Validating transaction (ID: "+newTransaction.ID+")...")
			valid, err := nxtblock.ValidatorValidateTransaction(newTransaction)
			if err != nil {
				nextutils.Error("%s", "Error: Transaction (ID: "+newTransaction.ID+") is not valid")
				nextutils.Error("Error: %v", err)
				nextutils.Error("%s", fmt.Sprintf("UTXO Database (formatted): %+v", nxtutxodb.GetUTXODatabase()))
				return
			}
			if !valid {
				nextutils.Error("%s", "Error: Transaction (ID: "+newTransaction.ID+") is not valid")
				nextutils.Error("%s", fmt.Sprintf("UTXO Database (formatted): %+v", nxtutxodb.GetUTXODatabase()))
				return
			}
			nextutils.Debug("%s", "Transaction (ID: "+newTransaction.ID+") is valid.")
		case "BLOCK":
			newBlock, err := nxtblock.GetBlockSender(newObject)
			if err != nil {
				nextutils.Error("Error: %v", err)
				return
			}

			nextutils.Debug("%s", "Validating block (ID: "+newBlock.Id+")...")
			valid, err := nxtblock.ValidatorValidateBlock(newBlock, blockdir, ruleset)
			if err != nil {
				nextutils.Error("%s", "Error: Block (ID: "+newBlock.Id+") is not valid")
				nextutils.Error("Error: %v", err)
				return
			}
			if !valid {
				nextutils.Error("%s", "Error: Block (ID: "+newBlock.Id+") is not valid")
				return
			}
			nextutils.Debug("%s", "Block (ID: "+newBlock.Id+") is valid.")
			fmt.Println("Block (ID: " + newBlock.Id + ") is valid.")
			nextutils.Debug("Saving block...")
			path := nxtblock.SaveBlock(newBlock, blockdir)
			nextutils.Debug("%s", "Block saved: "+path)
			nextutils.Debug("Updating UTXO database...")
			nxtblock.DeleteBlockUTXOs(newBlock.Transactions)
			nxtblock.ConvertBlockToUTXO(newBlock)
			adjustDifficulty()
			nextutils.Debug("UTXO database updated.")
		default:
			nextutils.Debug("%s", "Unknown new object: "+newObject)
		}
	default:
		nextutils.Debug("%s", "Unknown event: "+event)
	}

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
	start(peer)
}

// * STARTUP * //
func startup(debug *bool) {
	nextutils.InitDebugger(*debug)
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
	if err := configmanager.SetItem("ruleset", nxtblock.RuleSet{
		Difficulty:      6,
		MaxTransactions: 10,
		Version:         0,
		InitialReward:   50000000000000,
	}, &config, true); err != nil {
		nextutils.Error("Error setting ruleset: %v", err)
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

	rulesetMap := config.Fields["ruleset"].(map[string]interface{})
	ruleset = nxtblock.RuleSet{
		Difficulty:      int(rulesetMap["Difficulty"].(float64)),
		MaxTransactions: int(rulesetMap["MaxTransactions"].(float64)),
		Version:         int(rulesetMap["Version"].(float64)),
		InitialReward:   int64(rulesetMap["InitialReward"].(float64)),
	}

	nextutils.PrintLogo("V "+version+" - (c) 2025 NXTCHAIN. All rights reserved.\n-> NODE APPLICATION", devmode)
}
