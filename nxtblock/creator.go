package nxtblock

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
