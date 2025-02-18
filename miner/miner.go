package main

import (
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
var tick int = 5
var minerWallet string
var minerCurrency string
var timeTargetMin float64 = 10

// * CONFIG * //
var config configmanager.Config
var ruleset nxtblock.RuleSet

// ruleset := nxtblock.RuleSet{
// 	Difficulty:      6,
// 	MaxTransactions: 10,
// 	Version:         0,
// 	InitialReward:   5000000000000,
// }

// * MAIN START * //
func main() {
	seedNode := flag.String("seednode", "", "Optional seed node IP address")
	debug := flag.Bool("debug", false, "Enable debug mode")
	minerWallet := flag.String("minerwallet", "", "Miner wallet address")
	if *minerWallet != "" {
		configmanager.SetItem("miner_wallet", *minerWallet, &config, true)
	}
	minerCurrency := flag.String("currency", "NXT", "Currency to mine")
	if *minerCurrency != "" {
		configmanager.SetItem("miner_currency", *minerCurrency, &config, true)
	}

	flag.Parse()

	startup(debug)
	createPeer(*seedNode)
}

// * MAIN * //
func start(peer *gonetic.Peer) {
	if !devmode {
		clitools.ClearScreen()
	}
	nextutils.NewLine()
	nextutils.Debug("%s", "Beginning miner main actions...")
	nextutils.Debug("%s", "Connection string: "+peer.GetConnString())
	nextutils.Info("%s", "Your connection string: "+peer.GetConnString())
	nextutils.NewLine()
	nextutils.Debug("%s", "Waiting for peers to sync...")
	fmt.Println("+- WAITING FOR PEERS TO SYNC -")
	for {
		if len(peer.GetConnectedPeers()) > 0 {
			nextutils.Debug("%s", "Starting syncronization...")
			nextutils.Debug("%s", "Syncing Blockchain...")
			syncUTXODB(peer)
			syncBlockchain(peer)
			nextutils.Debug("%s", "Syncronization complete.")
			fmt.Println("+- SYNC COMPLETE -")
			break
		}
		time.Sleep(1 * time.Second)

	}

	var miningInProgress bool

	go func() {
		for {
			if !miningInProgress && len(nxtblock.GetAllTransactionsFromPool()) > 0 {
				miningInProgress = true
				nextutils.NewLine()
				nextutils.Debug("%s", "Mining new block...")
				fmt.Printf("Mining new block (Transactions in mempool: %d)\n", len(nxtblock.GetAllTransactionsFromPool()))

				start := time.Now()

				// genesisBlock := nxtblock.Block{
				// 	Id:           "0",
				// 	Timestamp:    0,
				// 	PreviousHash: "GENESIS",
				// 	Hash:         "GENESIS",
				// 	Data:         "GENESIS BLOCK",
				// 	Nonce:        0,
				// 	Transactions: []nxtblock.Transaction{},
				// 	Ruleset:      ruleset,
				// 	Currency:     "NXT",
				// 	BlockHeight:  0,
				// }

				latestBlock, err := nxtblock.GetLatestBlock(blockdir, true)
				if err != nil {
					nextutils.Error("Error getting latest block: %v", err)
					return
				}

				transactionMap := nxtblock.GetAllTransactionsFromPool()
				transactions := make([]nxtblock.Transaction, 0, len(transactionMap))
				for _, tx := range transactionMap {
					transactions = append(transactions, tx)
				}
				allblocks, err := nxtblock.GetAllBlocks(blockdir)
				if err != nil {
					nextutils.Error("Error getting all blocks: %v", err)
					return
				}
				if len(allblocks)%10 == 0 {
					adjustDifficulty()
				} else {
					nextutils.Debug("%s", "No need to adjust difficulty")
				}

				if len(transactions) > 0 {
					fmt.Println("+- Mapped transactions: ", transactions)
				}

				// * Create block
				newBlock, err := nxtblock.NewBlock(transactions, ruleset, minerWallet, minerCurrency, "I love NXT", latestBlock)
				if err != nil {
					nextutils.Error("Error creating new block: %v", err)
					continue
				}

				elapsed := time.Since(start)
				fmt.Printf("\n-- Done! (%s) - %s\n", elapsed, newBlock.Hash)

				// * Validate block
				nextutils.Debug("%s", "Validating new block...")
				_, err = nxtblock.ValidatorValidateBlock(*newBlock, blockdir, ruleset)
				if err != nil {
					nextutils.Error("Error validating block: %v", err)
					continue
				}

				fmt.Printf("\n[+] BLOCK IS VALID | YOU'VE EARNED %f NXT (%d)\n", nxtblock.ConvertAmount(newBlock.HeadTransactions[0].Outputs[0].Amount), newBlock.HeadTransactions[0].Outputs[0].Amount)
				//show the user why the blockreward is what it is
				fmt.Printf("-\tBlock reward: %d\n", nxtblock.CalculateBlockReward(ruleset.InitialReward, int64(newBlock.BlockHeight)))
				fmt.Printf("-\tBlock fee: %d\n", nxtblock.CalculateBlockFee(newBlock.Transactions))
				fmt.Printf("-\tBlock reward + fee: %d\n", nxtblock.CalculateBlockReward(ruleset.InitialReward, int64(newBlock.BlockHeight))+nxtblock.CalculateBlockFee(newBlock.Transactions))
				fmt.Printf("-\tWhat you received: %d\n", newBlock.HeadTransactions[0].Outputs[0].Amount)
				// * Update UTXO database
				nxtblock.DeleteBlockUTXOs(newBlock.Transactions)
				nxtblock.ConvertBlockToUTXO(*newBlock)

				// * Save block
				path := nxtblock.SaveBlock(*newBlock, blockdir)
				if path == "" {
					nextutils.Error("%s", "Error saving block")
					continue
				}

				fmt.Println("+- Block saved: " + path)

				// * Broadcast block
				blockStr, err := nxtblock.PrepareBlockSender(*newBlock)
				if err != nil {
					nextutils.Error("Error: %v", err)
					continue
				}

				peer.Broadcast("NEW_BLOCK_" + blockStr)
				for _, tx := range transactions {
					nxtblock.RemoveTransactionFromPool(tx)
				}
				miningInProgress = false
			}
			time.Sleep(time.Duration(tick) * time.Second)
		}
	}()

	// ~ MINER MAIN ACTIONS & LOG ~ //
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
				syncUTXODB(peer)
				syncBlockchain(peer)
				nextutils.Debug("%s", "Syncronization complete.")
				fmt.Println("+- SYNC COMPLETE -")
			} else if strings.HasPrefix(input, "$restart") {
				start(peer)
			} else if strings.HasPrefix(input, "$validate") {
				blocks, err := nxtblock.GetAllBlocks(blockdir)
				if err != nil {
					nextutils.Error("Error getting all blocks: %v", err)
					continue
				}
				for _, block := range blocks {
					nextutils.Debug("Validating block: %s P: %s", block.Hash, block.PreviousHash)
					_, err := nxtblock.ValidatorValidateBlock(block, blockdir, ruleset)
					if err != nil {
						nextutils.Error("Error validating block: %v", err)
						continue
					}
					nextutils.Debug("Block is valid.")
				}
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
			peer.Broadcast("RESPONSE_BLOCKHEIGHT_" + strconv.Itoa(blockHeight))
			nextutils.Debug("%s", "[+] Sent block height to: "+requester+" ("+strconv.Itoa(blockHeight)+")")
		} else if isBlock := strings.HasPrefix(event_body, "BLOCK_"); isBlock {
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
			peer.Broadcast("RESPONSE_BLOCK_" + blockStr)
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
			peer.Broadcast("RESPONSE_UTXODB_" + utxoDBStr)
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

			if totalResponses == 1 {
				go func() {
					time.Sleep(5 * time.Second)
					if totalResponses < remainingBlockHeights {
						nextutils.Debug("Timeout reached, proceeding with available responses")
						selectedHeight := getMostFrequentBlockHeight()
						nextutils.Debug("Selected block height for sync: %d", selectedHeight)
						startBlockchainSync(selectedHeight, peer)
					}
				}()
			}

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

			go func() {
				time.Sleep(5 * time.Second)
				if totalUDBresponses < remainingDBs {
					nextutils.Debug("Timeout reached, proceeding with available UTXO DB responses")
					selectedDB := getMostFrequentUTXODB()
					nextutils.Debug("Selected UTXO DB for sync: %v", selectedDB)
					nxtutxodb.SetUTXODatabase(selectedDB)
				}
			}()

			if totalUDBresponses >= remainingDBs { // * ALL RESPONSES FOR A VALID * //
				selectedDB := getMostFrequentUTXODB()
				nextutils.Debug("Selected UTXO DB for sync: %v", selectedDB)
				nxtutxodb.SetUTXODatabase(selectedDB)
			}
		} else if strings.HasPrefix(event_body, "BLOCK_") {
			parts := strings.SplitN(event_body, "_", 2)
			if len(parts) < 2 {
				nextutils.Error("%s", "Invalid event body format: "+event_body)
				return
			}
			respObject := parts[1]
			newBlock, err := nxtblock.GetBlockSender(respObject)
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
			nextutils.Debug("%s", "Block (ID: "+newBlock.Id+") is valid. Saving block...")
			path := nxtblock.SaveBlock(newBlock, blockdir)
			nextutils.Debug("%s", "Block saved: "+path)
			nextutils.Debug("Updating UTXO database...")
			nxtblock.DeleteBlockUTXOs(newBlock.Transactions)
			nxtblock.ConvertBlockToUTXO(newBlock)

			allblocks, err := nxtblock.GetAllBlocks(blockdir)
			if err != nil {
				nextutils.Error("Error getting all blocks: %v", err)
				return
			}
			if len(allblocks)%10 == 0 {
				adjustDifficulty()
			} else {
				nextutils.Debug("%s", "No need to adjust difficulty")
			}
			nextutils.Debug("UTXO database updated.")
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
			//! TEST
			nxtutxodb.AddUTXO("1", 0, 100000000000000, "rpiZNDkFnb7f5CnYTnoASqHHUSt1Jpn4dJLSqH4tLSw", 1, false)
			//! -----
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
			nxtblock.AddTransactionToPool(newTransaction)
			fmt.Println("[+] Added transaction: #" + newTransaction.ID + " to the mempool")
			nextutils.Debug("%s", "Mempool size: "+strconv.Itoa(len(nxtblock.GetAllTransactionsFromPool())))
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
			nextutils.Debug("UTXO database updated.")
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
	if selectedHeight > localHeight {
		remainingBlockHeights = selectedHeight - localHeight
		total := float64(remainingBlockHeights)

		for i := localHeight; i < selectedHeight; i++ {
			progress := float64(i-localHeight) / total * 100
			nextutils.Info("\rSynchronizing blocks: %.1f%% (%d/%d)", progress, i-localHeight, remainingBlockHeights)
			peer.Broadcast("RGET_BLOCK_" + strconv.Itoa(i) + "_" + peer.GetConnString())
		}
		nextutils.Info("\rSynchronizing blocks: 100.0%% (%d/%d)\n", remainingBlockHeights, remainingBlockHeights)
		nextutils.Debug("Block synchronization requests completed")
	} else {
		nextutils.Debug("Local blockchain is ahead or equal to network height. No sync needed.")
	}
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
	valid := avgTime < timeTargetMin || avgTime > timeTargetMin
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
	configmanager.SaveConfig(config)
	nextutils.Debug("Difficulty should %s", direction)
	nextutils.Debug("New difficulty: %d", ruleset.Difficulty)
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
	seedNodes := []string{}
	if seedNodesVal, exists := config.Fields["seed_nodes"]; exists && seedNodesVal != nil {
		if seedNodesInterface, ok := seedNodesVal.([]interface{}); ok {
			seedNodes = make([]string, len(seedNodesInterface))
			for i, v := range seedNodesInterface {
				seedNodes[i] = v.(string)
			}
		}
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
	if err := configmanager.SetItem("tick", float64(5), &config, true); err != nil {
		nextutils.Error("Error setting block_dir: %v", err)
		return
	}
	if err := configmanager.SetItem("default_port", "5012", &config, true); err != nil {
		nextutils.Error("Error setting default_port: %v", err)
		return
	}
	if err := configmanager.SetItem("miner_wallet", "", &config, true); err != nil {
		nextutils.Error("Error setting miner_wallet: %v", err)
		return
	}
	if err := configmanager.SetItem("miner_currency", "NXT", &config, true); err != nil {
		nextutils.Error("Error setting miner_currency: %v", err)
		return
	}
	if err := configmanager.SetItem("max_connections", float64(10), &config, true); err != nil {
		nextutils.Error("Error setting max_connections: %v", err)
		return
	}
	if err := configmanager.SetItem("ruleset", nxtblock.RuleSet{
		Difficulty:      6,
		MaxTransactions: 10,
		Version:         0,
		InitialReward:   5000000000000,
	}, &config, true); err != nil {
		nextutils.Error("Error setting ruleset: %v", err)
		return
	}
	nextutils.Debug("%s", "Config applied.")
	for key, value := range config.Fields {
		nextutils.Debug("- %s = %v", key, value)
	}

	if config.Fields["block_dir"] != nil {
		blockdir = config.Fields["block_dir"].(string)
	}
	if config.Fields["tick"] != nil {
		tick = int(config.Fields["tick"].(float64))
	}
	if config.Fields["miner_wallet"] != nil {
		minerWallet := config.Fields["miner_wallet"].(string)
		if minerWallet != "" {
			nextutils.Debug("%s", "Miner wallet: "+minerWallet)
		} else {
			nextutils.Debug("%s", "No miner wallet set.")
		}
	}
	if config.Fields["miner_currency"] != nil {
		minerCurrency = config.Fields["miner_currency"].(string)
		if minerCurrency != "" {
			nextutils.Debug("%s", "Miner currency: "+minerCurrency)
		} else {
			nextutils.Debug("%s", "No miner currency set.")
		}
	}

	ruleset = config.Fields["ruleset"].(nxtblock.RuleSet)

	nextutils.PrintLogo("V "+version+" - (c) 2025 NXTCHAIN. All rights reserved.\n-> MINER APPLICATION", devmode)
}
