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
	"sort"
	"strconv"
	"strings"
	"time"
)

// * GLOBAL VARS * //

var version string = "0.0.0"
var devmode bool = true
var inputs []nxtutxodb.UTXO
var requestedInputWallet string
var walletdir string = "wallets"
var balanceJSON string
var requestedWalletAddr string
var transactionsJSON string

// * CONFIG * //
var config configmanager.Config

// * MAIN START * //

func main() {
	// walletDir := "wallets"
	// if config.Fields["wallets"] != nil {
	// 	walletDir = config.Fields["wallets"].(string)
	// }

	seednode := flag.String("seednode", "", "Optional seed node IP address")
	debug := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	startup(debug)
	createPeer(*seednode)
}

// * MAIN * //
func start(Peer *gonetic.Peer, msg ...string) {
	if !devmode {
		clitools.ClearScreen()
	}
	nextutils.NewLine()
	nextutils.Debug("%s", "Starting wallet...")
	nextutils.Debug("%s", "Connection string: "+Peer.GetConnString())
	nextutils.Info("%s", "Your connection string: "+Peer.GetConnString())

	// ! Wallet's seed nodes are not necessary, you could also use an API that is connected to the network, aslong you can send transactions

	// ~ WALLET MAIN ACTIONS & LOG ~ //
	// ? Send, Receive, Balance, Transactions, etc.
	// ? REMEMBER: Also use the change in the UTXOs

	// * WALLET MENU * //
	nextutils.NewLine()
	nextutils.Debug("%s", "Wallet menu")

	time.Sleep(1 * time.Second)

	if len(msg) > 0 && msg[0] != "" {
		fmt.Println(msg[0])
	}
	fmt.Println("OPTIONS:")
	fmt.Println("1. Send")
	fmt.Println("2. Receive")
	fmt.Println("3. Balance")
	fmt.Println("4. Transactions")
	fmt.Println("5. Create Wallet")
	fmt.Println("6. Exit")

	fmt.Print("> ")
	var option int
	fmt.Scanln(&option)

	nextutils.Debug("%s", "Option: "+strconv.Itoa(option))

	switch option {
	case 1:
		nextutils.NewLine()
		nextutils.Debug("%s", "PREPARING TO SEND TRANSACTION...")

		wallets, err := nxtblock.GetAllWallets(walletdir)
		if err != nil {
			nextutils.Error("Error getting wallets: %v", err)
			return
		}
		var walletAddresses []string
		for _, wallet := range wallets {
			walletAddresses = append(walletAddresses, nxtblock.GenerateWalletAddress([]byte(wallet.PublicKey)))
		}

		fmt.Println("CHOOSE A WALLET TO SEND:")
		for i, addr := range walletAddresses {
			fmt.Printf("%d | %s\n", i+1, addr)
		}
		fmt.Print("> ")
		var walletIndex int
		fmt.Scanln(&walletIndex)
		walletIndex--
		if walletIndex < 0 || walletIndex >= len(walletAddresses) {
			fmt.Println("Invalid wallet index")
			start(Peer)
			return
		}
		fmt.Println("SET: " + walletAddresses[walletIndex])
		fmt.Print("TO: ") // Wallet receiving address
		var to string
		fmt.Scanln(&to)
		fmt.Print("AMOUNT: ") // Amount to send
		var amount float64    // * Amount * 1.000.000.000.000
		fmt.Scanln(&amount)
		fmt.Print("FEE (OPTIONAL > 0) IN % 0-100: ") // Fee percentage
		var fee int64                                // * Fee PERCENTAGE *
		fmt.Scanln(&fee)

		walletAddr := walletAddresses[walletIndex]
		fmt.Println("=====================================")
		fmt.Println("FROM:              ", walletAddr)
		fmt.Println("TO:                ", to)
		fmt.Println("AMOUNT IN NXT:     ", nxtblock.ConvertAmountBack(amount))
		fmt.Println("AMOUNT RAW:        ", amount)
		fmt.Println("FEE:               ", fmt.Sprintf("%d%%", fee))
		fmt.Printf("FEE IN NXT:         %d\n", int64(float64(nxtblock.ConvertAmountBack(amount))*(float64(fee)/100)))
		fmt.Println("=====================================")
		nextutils.Debug("%s", "Transaction details: "+fmt.Sprintf("FROM: %s, TO: %s, AMOUNT: %f, FEE: %d", walletAddr, to, amount, fee))

		nextutils.Debug("%s", "Requesting unspent transaction outputs (UTXOs) for wallet: "+walletAddr)
		getInputs(Peer, walletAddr)

		timeoutChan := time.After(30 * time.Second)
		select {
		case <-timeoutChan:
			fmt.Println("Timeout: Failed to retrieve wallet inputs after 30 seconds")
			start(Peer)
			return
		case <-time.After(100 * time.Millisecond):
			if len(inputs) > 0 {
				break
			}
		}

		for len(inputs) == 0 {
			time.Sleep(1 * time.Second)
		}

		nextutils.Debug("%s", "Received inputs: "+fmt.Sprintf("%v", inputs))

		feeAmount := (nxtblock.ConvertAmountBack(amount) * fee) / 100
		totalNeeded := nxtblock.ConvertAmountBack(amount) + feeAmount
		var selectedInputs []nxtutxodb.UTXO
		var totalAmount int64 = 0

		sort.Slice(inputs, func(i, j int) bool {
			return inputs[i].Amount > inputs[j].Amount
		})

		for _, input := range inputs {
			selectedInputs = append(selectedInputs, input)
			totalAmount += input.Amount
			if totalAmount >= totalNeeded {
				break
			}
		}

		if totalAmount < totalNeeded {
			fmt.Printf("ERROR: Insufficient funds (%.8f NXT required, have %.8f NXT)\n",
				float64(totalNeeded),
				float64(totalAmount))
			start(Peer)
			return
		}

		change := totalAmount - totalNeeded
		if change > 0 {
			fmt.Printf("Change amount: %.8f NXT\n", float64(change))
		}

		fmt.Printf("Selected %d inputs for transaction\n", len(selectedInputs))
		fmt.Printf("Inputs: %v\n", selectedInputs)

		var tInputs []nxtblock.TInput
		var tOutputs []nxtblock.TOutput

		tOutput := nxtblock.CreateTransactionOutput(0, nxtblock.ConvertAmountBack(amount), to)
		tOutputs = append(tOutputs, tOutput)

		if change > 0 {
			changeOutput := nxtblock.CreateTransactionOutput(1, change, walletAddr)
			tOutputs = append(tOutputs, changeOutput)
		}

		// * ASK FOR CONFIRMATION * //
		fmt.Println("CONFIRM TRANSACTION? (Y/n)")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "Y" && confirm != "y" && confirm != "" {
			start(Peer)
			return
		}

		tx := nxtblock.PrepareTransaction(tOutputs)

		for _, input := range selectedInputs {
			wallet, err := nxtblock.LoadWallet(walletAddr, walletdir)
			if err != nil {
				nextutils.Error("Error loading wallet: %v", err)
				return
			}
			tInput := nxtblock.CreateTransactionInput(input.Txid, input.Index, []byte(wallet.PrivateKey), []byte(wallet.PublicKey), tx.Hash)
			tInputs = append(tInputs, tInput)
		}

		tx.Inputs = tInputs

		// txJSON, _ := json.Marshal(tx)
		// fmt.Printf("Transaction JSON: %s\n", string(txJSON))

		// * RESET INPUTS * //
		inputs = nil

		// * SEND TRANSACTION * //

		txs, err := nxtblock.PrepareTransactionSender(tx)
		if err != nil {
			nextutils.Error("Error preparing transaction sender: %v", err)
			return
		}
		Peer.Broadcast("NEW_TRANSACTION_" + txs)

		start(Peer)

	case 2:
		fmt.Println("TO RECEIVE CRYPTO, YOU NEED TO SHARE YOUR WALLET ADDRESS")
		fmt.Println("YOUR WALLETS:")
		wallets, err := nxtblock.GetAllWallets(walletdir)
		if err != nil {
			nextutils.Error("Error getting wallets: %v", err)
			return
		}
		for _, wallet := range wallets {
			fmt.Println("- " + nxtblock.GenerateWalletAddress(wallet.PublicKey))
		}
		start(Peer)
	case 3:
		wallets, err := nxtblock.GetAllWallets(walletdir)
		if err != nil {
			nextutils.Error("Error getting wallets: %v", err)
			return
		}
		var walletAddresses []string
		for _, wallet := range wallets {
			walletAddresses = append(walletAddresses, nxtblock.GenerateWalletAddress([]byte(wallet.PublicKey)))
		}
		balanceJSON = ""
		fmt.Println("CHOOSE A WALLET TO SEND:")
		for i, addr := range walletAddresses {
			fmt.Printf("%d | %s\n", i+1, addr)
		}
		fmt.Print("> ")
		var walletIndex int
		fmt.Scanln(&walletIndex)
		walletIndex--
		if walletIndex < 0 || walletIndex >= len(walletAddresses) {
			fmt.Println("Invalid wallet index")
			start(Peer)
			return
		}
		nextutils.Debug("%s", "Selected wallet: "+walletAddresses[walletIndex])
		fmt.Println("SET: " + walletAddresses[walletIndex])
		// ? Ask the node for balance
		requestedWalletAddr = walletAddresses[walletIndex]
		Peer.Broadcast("RGET_BALANCE_" + walletAddresses[walletIndex] + "_" + Peer.GetConnString())
		fmt.Println("Requesting balance for wallet: " + walletAddresses[walletIndex] + "... (Please wait)")

		for balanceJSON == "" {
			time.Sleep(420 * time.Millisecond)
		}

		rutxos, err := RetrieveInputs(balanceJSON)
		if err != nil {
			nextutils.Error("Error retrieving UTXOs: %v", err)
			return
		}
		amount := nxtblock.ConvertAmount(CalculateBalance(rutxos))
		start(Peer, fmt.Sprintf("%s | BALANCE: %f NXT", walletAddresses[walletIndex], amount))

	case 4:
		wallets, err := nxtblock.GetAllWallets(walletdir)
		if err != nil {
			nextutils.Error("Error getting wallets: %v", err)
			return
		}
		var walletAddresses []string
		for _, wallet := range wallets {
			walletAddresses = append(walletAddresses, nxtblock.GenerateWalletAddress([]byte(wallet.PublicKey)))
		}
		balanceJSON = ""
		fmt.Println("CHOOSE A WALLET TO SEE TRANSACTIONS:")
		for i, addr := range walletAddresses {
			fmt.Printf("%d | %s\n", i+1, addr)
		}
		fmt.Print("> ")
		var walletIndex int
		fmt.Scanln(&walletIndex)
		walletIndex--
		if walletIndex < 0 || walletIndex >= len(walletAddresses) {
			fmt.Println("Invalid wallet index")
			start(Peer)
			return
		}
		nextutils.Debug("%s", "Selected wallet: "+walletAddresses[walletIndex])
		fmt.Println("SET: " + walletAddresses[walletIndex])
		// ? Ask the node for transactions
		requestedWalletAddr = walletAddresses[walletIndex]
		Peer.Broadcast("RGET_TRANSACTIONS_" + walletAddresses[walletIndex] + "_" + Peer.GetConnString())
		fmt.Println("Requesting transactions for wallet: " + walletAddresses[walletIndex] + "... (Please wait)")

		for transactionsJSON == "" {
			time.Sleep(420 * time.Millisecond)
		}
		transactions, err := RetrieveTransactions(transactionsJSON)
		if err != nil {
			nextutils.Error("Error retrieving TXs: %v", err)
			start(Peer, "No transactions found")
		}

		formattedTransactions := make([]string, 0)
		for _, tx := range transactions {
			formattedTransactions = append(formattedTransactions, fmt.Sprintf("TXID: %s | %s -> %s | AMOUNT: %f NXT", tx.Hash, tx.Inputs[0].PublicKey, tx.Outputs[0].ReceiverAddr, nxtblock.ConvertAmount(tx.Outputs[0].Amount)))
		}

		start(Peer, formattedTransactions...)
	case 5:
		fmt.Println("CREATE WALLET")
		nextutils.NewLine()
		nextutils.Debug("%s", "Creating wallet...")
		fmt.Println("Enter some random text to create a wallet:")
		var text string
		fmt.Scanln(&text)
		nextutils.Debug("%s", "SEED Text: "+text)
		wallet := nxtblock.CreateWallet([]byte(text))
		walletPath := nxtblock.SaveWallet(wallet, walletdir)
		if walletPath == "" {
			nextutils.Error("%s", "Error creating wallet")
			return
		}
		nextutils.Debug("%s", "Wallet created: "+walletPath)
		fmt.Println("Wallet created: " + walletPath)
		start(Peer)
	case 6:
		fmt.Println("EXIT")
		Peer.Stop()
		return
	default:
		clitools.ClearScreen()
		start(Peer)
	}
}

// * GET INPUTS FOR WALLET * //
// ? Can be replaced by a API or Node or any other way to get the inputs
func getInputs(Peer *gonetic.Peer, walletAddr string) {
	nextutils.Debug("%s", "Requesting inputs for wallet: "+walletAddr)
	requestedInputWallet = walletAddr
	Peer.Broadcast("RGET_INPUTS_" + walletAddr + "_" + Peer.GetConnString())
}

// * CALCULATE BALANCE * //

func CalculateBalance(inputs []nxtutxodb.UTXO) int64 {
	var balance int64
	for _, input := range inputs {
		balance += input.Amount
	}
	return balance
}

// * RETRIEVE INPUTS FROM UTXO JSON * //

func RetrieveInputs(inputsJson string) ([]nxtutxodb.UTXO, error) {
	utxos, err := nxtblock.RetrieveUTXOFromJSON(inputsJson)
	if err != nil {
		return nil, err
	}
	return utxos, nil
}

// * RETRIEVE TRANSACTION FROM TX JSON * //

func RetrieveTransactions(inputsJson string) ([]nxtblock.Transaction, error) {
	txs, err := nxtblock.RetrieveTransactionsFromJSON(inputsJson)
	if err != nil {
		return nil, err
	}
	return txs, nil
}

// * PEER OUTPUT HANDLER * //
func handleEvents(event string) {
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
	case "RESPONSE":
		nextutils.Debug("%s", "[RESPONSE] "+event_body)
		if strings.HasPrefix(event_body, "INPUTS_") {
			inputsJson := strings.TrimPrefix(event_body, "INPUTS_")
			var err error
			utxos, err := nxtblock.RetrieveUTXOFromJSON(inputsJson)
			if err != nil {
				nextutils.Error("Error retrieving UTXOs: %v", err)
				return
			}

			validUtxos := make([]nxtutxodb.UTXO, 0)
			for _, utxo := range utxos {
				if utxo.PubKey == requestedInputWallet {
					validUtxos = append(validUtxos, utxo)
				}
			}

			inputs = validUtxos
		}
		if strings.HasPrefix(event_body, "BALANCE_") {
			balance := strings.TrimPrefix(event_body, "BALANCE_")
			parts := strings.SplitN(balance, "_", 2)
			if strings.TrimSpace(parts[1]) == strings.TrimSpace(requestedWalletAddr) {
				nextutils.Debug("%s", "Balance: "+parts[1])
				balanceJSON = parts[0]
			}
		}
		if strings.HasPrefix(event_body, "TRANSACTIONS_") {
			transactions := strings.TrimPrefix(event_body, "TRANSACTIONS_")
			parts := strings.Split(transactions, "_")
			if strings.TrimSpace(parts[1]) == strings.TrimSpace(requestedWalletAddr) {
				nextutils.Debug("%s", "Transactions: "+parts[1])
				transactionsJSON = parts[0]
			}
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
func startup(debug *bool) {
	nextutils.InitDebugger(*debug)
	nextutils.NewLine()
	nextutils.Debug("Starting wallet...")
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
	if err := configmanager.SetItem("wallet_dir", "wallets", &config, true); err != nil {
		nextutils.Error("Error setting wallet_dir: %v", err)
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

	if config.Fields["wallet_dir"] != nil {
		walletdir = config.Fields["wallet_dir"].(string)
	}

	nextutils.PrintLogo("V "+version+" - (c) 2025 NXTCHAIN. All rights reserved.\n-> WALLET APPLICATION", devmode)
}
