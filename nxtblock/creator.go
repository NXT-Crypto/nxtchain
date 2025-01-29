package nxtblock

import "math"

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

func NewBlock(transactions []Transaction, ruleset RuleSet, minerAddr string) {

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
