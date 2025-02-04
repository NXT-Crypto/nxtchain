package nxtblock

import (
	"crypto/sha256"
	"fmt"
	"math"
	"math/rand"
	"nxtchain/nextutils"
	"nxtchain/nxtutxodb"
	"sync"
	"time"
)

type RuleSet struct {
	Difficulty      int
	MaxTransactions int
	Version         int
	InitialReward   int64
}

// ! Block struct Marked with "+" are the ones in the hash

type Block struct {
	Id               string        // ID des Blocks 												+
	Timestamp        int64         // Zeitstempel des Blocks										+
	PreviousHash     string        // Hash des vorherigen Blocks									+
	Hash             string        // Hash des Blocks
	Data             string        // Daten des Blocks												+
	TransactionHash  string        // Hashes der Transaktionen vereint								+
	Nonce            int64         // Zufallszahl die den Blockhash erzeugt
	Transactions     []Transaction // Transaktionen des Blocks										+
	HeadTransactions []Transaction // Head-Transaktionen des Blocks (Miner-Transaktionen)			+
	Ruleset          RuleSet       // Regeln des Blocks												+
	Currency         string        // Währung des Blocks (aus welcher währung basiert der Block)	+
	BlockHeight      int           // Höhe des Blocks
}

func NewBlock(transactions []Transaction, ruleset RuleSet, minerAddr string, currency string, data string, lastblock Block) (*Block, error) {
	nextutils.NewLine()
	nextutils.Debug("Beginning block creation...")
	nextutils.Debug("- Transactions: %v", transactions)
	nextutils.Debug("- Difficulty: %v", ruleset.Difficulty)
	nextutils.Debug("- Max Transactions: %v", ruleset.MaxTransactions)
	nextutils.Debug("- Version: %v", ruleset.Version)
	nextutils.Debug("- Miner Address: %v", minerAddr)
	nextutils.Debug("- Currency: %v", currency)
	nextutils.Debug("- Last Block Hash: %v", lastblock.Hash)
	nextutils.Debug("- Data: %v", data)
	nextutils.NewLine()
	nextutils.Debug("Validating transactions...")
	for _, tx := range transactions {
		nextutils.Debug("- Transaction: %v", tx)
	}

	// * ABLAUF DER BLOCKERSTELLUNG * //
	// ? 1. Fee berechnen & als Belohnung speichern
	// ? 2. Transactionhash berechnen & MaxTransactions prüfen
	// ? 3. HeadTransaction erstellen (aus Belohnung)
	// ? 4. Blockhash berechnen (Proof of Work mit difficulty aus dem ruleset)

	// ? 5. Block validieren

	// * 1. FEE BERECHNEN * //
	blockFee := CalculateBlockFee(transactions)
	nextutils.Debug("Block Fee: %v", blockFee)

	// * 2. TRANSACTIONHASH BERECHNEN * //
	transactionHash := CalculateTransactionHash(transactions)

	// * 2.1 MAXTRANSACTIONS PRÜFEN * //
	maxTransactions := ruleset.MaxTransactions
	if len(transactions) > maxTransactions {
		return nil, fmt.Errorf("too many transactions in block: %d > %d", len(transactions), maxTransactions)
	}

	// * 3. HEADTRANSACTION ERSTELLEN * //
	headTransaction := CreateTransactionHeader(minerAddr, int64(blockFee)+CalculateBlockReward(ruleset.InitialReward, int64(lastblock.BlockHeight+1)))

	timestamp := time.Now().Unix()

	blockID := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d%s%v%v%v",
		timestamp,
		lastblock.Hash,
		blockFee,
		transactionHash,
		headTransaction.Hash))))

	// * 4. BLOCKHASH BERECHNEN * //
	workers := 1
	var wg sync.WaitGroup
	hashChannel := make(chan string, workers)
	nonceChannel := make(chan int64, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		strategy := "ascending"
		go CreateBlockHash(i, workers, blockID, timestamp, lastblock.Hash, data, transactionHash, ruleset, currency, hashChannel, nonceChannel, &wg, strategy)
	}
	wg.Wait()
	close(hashChannel)
	close(nonceChannel)

	blockhash := <-hashChannel
	nonce := <-nonceChannel

	newBlock := &Block{
		Id:               blockID,
		Timestamp:        timestamp,
		PreviousHash:     lastblock.Hash,
		Hash:             blockhash,
		Data:             data,
		TransactionHash:  transactionHash,
		Nonce:            nonce,
		Transactions:     transactions,
		HeadTransactions: []Transaction{headTransaction},
		Ruleset:          ruleset,
		Currency:         currency,
		BlockHeight:      lastblock.BlockHeight + 1,
	}

	return newBlock, nil

}

// * CALCULATE BLOCK REWARD * //
// ? initialReward: Anfangsbelohnung
// ? decayRate: Zerfallsrate
// ? n: Anzahl der Blöcke
func CalculateBlockReward(initialReward int64, n int64) int64 {
	reward := float64(initialReward) * math.Exp(
		-(0.2420 * math.Log(float64(n+1)) * (float64(n) / float64(n+10))),
	)

	return int64(reward)
}

// * CALCULATE BLOCK FEE * //

func CalculateTransactionFee(tx Transaction) (int64, error) {
	var totalInput, totalOutput int64

	for _, in := range tx.Inputs {
		amount, err := nxtutxodb.GetUTXOAmount(in.Txid, in.Index)
		if err != nil {
			return 0, fmt.Errorf("error retrieving UTXO: %v", err)
		}
		totalInput += amount
	}

	for _, out := range tx.Outputs {
		totalOutput += out.Amount
	}

	return totalInput - totalOutput, nil
}

// * CALCULATE BLOCK FEE * //

func CalculateBlockFee(transactions []Transaction) int64 {
	var totalFee int64

	for _, tx := range transactions {
		fee, err := CalculateTransactionFee(tx)
		if err != nil {
			nextutils.Debug("Skipping transaction due to error: %v", err)
			continue
		}
		totalFee += fee
	}

	return totalFee
}

// * CALCULATE TRANSACTION HASH * //

func CalculateTransactionHash(transactions []Transaction) string {
	if len(transactions) == 0 {
		return fmt.Sprintf("%x", sha256.Sum256([]byte("")))
	}

	var leaves []string
	for _, tx := range transactions {
		leaves = append(leaves, tx.Hash)
	}

	for len(leaves) > 1 {
		if len(leaves)%2 == 1 {
			leaves = append(leaves, leaves[len(leaves)-1])
		}
		var temp []string
		for i := 0; i < len(leaves); i += 2 {
			combined := leaves[i] + leaves[i+1]
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(combined)))
			temp = append(temp, hash)
		}
		leaves = temp
	}

	return leaves[0]
}

// * CREATE BLOCK HASH * //

func CreateBlockHash(workerID int, workers int, Id string, timestamp int64, previousHash string, data string, transactionHash string, ruleset RuleSet, currency string, hashChannel chan string, nonceChannel chan int64, wg *sync.WaitGroup, strategy string) {
	defer wg.Done()

	var hash string
	var nonce int64
	maxNonce := int64(math.MaxInt64)

	switch strategy {
	case "random":
		r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))
		nonce = r.Int63n(maxNonce)

	case "ascending":
		nonce = int64(workerID) * (maxNonce / int64(workers))
		if nonce > maxNonce {
			nonce = maxNonce
		}

	case "descending":
		nonce = maxNonce - int64(workerID)*((maxNonce)/int64(workers))
		if nonce < 0 {
			nonce = 0
		}
	}

	startTime := time.Now()
	for {
		hashStr := fmt.Sprintf("%s%d%s%s%s%d%s",
			Id,
			timestamp,
			previousHash,
			data,
			transactionHash,
			nonce,
			currency,
		)

		hashBytes := sha256.Sum256([]byte(hashStr))
		hash = fmt.Sprintf("%x", hashBytes)

		leadingZeros := 0
		for i := 0; i < len(hash) && hash[i] == '0'; i++ {
			leadingZeros++
		}

		if nonce >= maxNonce {
			break
		}

		if nonce%50000 == 0 {
			elapsed := time.Since(startTime)
			fmt.Printf("\rWorker [%d]   │   Nonce: %d   │   Hash: %s   │   Time: %8v   │   Target: %2d   │   Current: %2d",
				workerID, nonce, hash, elapsed.Round(time.Millisecond), ruleset.Difficulty, leadingZeros)
		}

		if leadingZeros >= ruleset.Difficulty {
			hashChannel <- hash
			nonceChannel <- nonce
			return
		}

		if strategy == "ascending" {
			nonce++
		} else if strategy == "descending" {
			nonce--
		} else if strategy == "random" {
			nonce = rand.Int63n(maxNonce)
		}
	}
}

// * VALIDATE BLOCK HASH * //

func ValidateBlockHash(block Block) bool {
	hashStr := fmt.Sprintf("%s%d%s%s%s%d%s",
		block.Id,
		block.Timestamp,
		block.PreviousHash,
		block.Data,
		block.TransactionHash,
		block.Nonce,
		block.Currency,
	)

	hashBytes := sha256.Sum256([]byte(hashStr))
	hash := fmt.Sprintf("%x", hashBytes)

	return hash == block.Hash
}
