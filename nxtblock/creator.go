package nxtblock

import (
	"fmt"
	"math"
	"nxtchain/nextutils"
	"nxtchain/nxtutxodb"
)

type RuleSet struct {
	difficulty      int
	maxTransactions int
	version         int
}

type Block struct {
	id               string
	timestamp        int64
	previousHash     string
	hash             string
	data             string
	transactionHash  string
	nonce            int
	transactions     []Transaction
	headTransactions []Transaction
	ruleset          RuleSet
	Currency         string
}

func NewBlock(transactions []Transaction, ruleset RuleSet, minerAddr string, currency string, lastblock Block) (*Block, error) {
	nextutils.NewLine()
	nextutils.Debug("Beginning block creation...")
	nextutils.Debug("- Transactions: %v", transactions)
	nextutils.Debug("- Difficulty: %v", ruleset.difficulty)
	nextutils.Debug("- Max Transactions: %v", ruleset.maxTransactions)
	nextutils.Debug("- Version: %v", ruleset.version)
	nextutils.Debug("- Miner Address: %v", minerAddr)
	nextutils.Debug("- Currency: %v", currency)
	nextutils.Debug("- Last Block Hash: %v", lastblock.hash)
	nextutils.NewLine()
	nextutils.Debug("Validating transactions...")
	for _, tx := range transactions {
		nextutils.Debug("- Transaction: %v", tx)
	}

	return &Block{
		id:               "",
		timestamp:        0,
		previousHash:     "",
		hash:             "",
		data:             "",
		transactionHash:  "",
		nonce:            0,
		transactions:     transactions,
		headTransactions: []Transaction{},
		ruleset:          ruleset,
		Currency:         currency,
	}, nil

}

// * CALCULATE BLOCK REWARD * //
// ? initialReward: Anfangsbelohnung
// ? decayRate: Zerfallsrate
// ? n: Anzahl der Blöcke
func CalculateBlockReward(n float64, initialReward float64) float64 {
	reward := initialReward * math.Exp(
		-(0.2420 * math.Log(n+1) * (n / (n + 10))),
	)

	return reward
}

// * CALCULATE BLOCK FEE * //

func CalculateTransactionFee(tx Transaction) (float64, error) {
	var totalInput, totalOutput float64

	for _, in := range tx.Inputs {
		amount, err := nxtutxodb.GetUTXOAmount(in.Txid, in.Index)
		if err != nil {
			return 0, fmt.Errorf("error retrieving UTXO: %v", err)
		}
		totalInput += float64(amount)
	}

	for _, out := range tx.Outputs {
		totalOutput += float64(out.Amount)
	}

	return totalInput - totalOutput, nil
}

// * CALCULATE BLOCK FEE * //

func CalculateBlockFee(transactions []Transaction) float64 {
	var totalFee float64

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
