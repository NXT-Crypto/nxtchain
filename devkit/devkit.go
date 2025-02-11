package main

import (
	"fmt"
	"math"
	"nxtchain/nxtblock"
	"sort"
	"time"
)

func main() {
	fmt.Println("NXTChain DevKit v0.1 - 2025\n__________________________")

	fmt.Print("OPTIONS:\n0. GEN GENESIS BLOCK\n1. GEN BLOCK\n2. Difficulty Adjustment\n\nEnter option: ")
	var option int
	fmt.Scanln(&option)

	switch option {
	case 0:
		GGB()
	case 1:
		GB()
	case 2:
		DFBC(10)
	default:
		fmt.Println("Invalid option")
	}
}
func GGB() {
	fmt.Println("Generating genesis block...")

	ruleset := nxtblock.RuleSet{
		Difficulty:      6,
		MaxTransactions: 10,
		Version:         0,
		InitialReward:   50000000000000,
	}
	genesisBlock := nxtblock.Block{
		Id:           "0",
		Timestamp:    0,
		PreviousHash: "GENESIS",
		Hash:         "GENESIS",
		Data:         "GENESIS BLOCK",
		Nonce:        0,
		Transactions: []nxtblock.Transaction{},
		Ruleset:      ruleset,
		Currency:     "NXT",
		BlockHeight:  0,
	}
	transactions := []nxtblock.Transaction{
		{
			ID:   "1",
			Hash: "1",
			Inputs: []nxtblock.TInput{
				{
					Txid:  "0",
					Index: 0,
				},
			},
			Outputs: []nxtblock.TOutput{
				{
					Index:        0,
					ReceiverAddr: "A",
					Amount:       10000000000000,
				},
				{
					Index:        1,
					ReceiverAddr: "B",
					Amount:       10000000000000,
				},
			},
		},
	}
	start := time.Now()
	block, err := nxtblock.NewBlock(transactions, ruleset, "0xMINER", "NXT", "DATA", genesisBlock)
	if err != nil {
		fmt.Println("Error creating genesis block")
		return
	}
	elapsed := time.Since(start)
	fmt.Printf("\n-- Done! (%s) - %s\n", elapsed, block.Hash)
	fmt.Println("Saved:", nxtblock.SaveBlock(*block, "blocks"))

}

func GB() {
	fmt.Println("Generating block...")

	ruleset := nxtblock.RuleSet{
		Difficulty:      6,
		MaxTransactions: 10,
		Version:         0,
		InitialReward:   50000000000000,
	}
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
	lblock, err := nxtblock.GetLatestBlock("blocks", false)
	if err != nil {
		fmt.Println("Error getting latest block")
		return
	}
	transactions := []nxtblock.Transaction{
		{
			ID:   "1",
			Hash: "1",
			Inputs: []nxtblock.TInput{
				{
					Txid:  "0",
					Index: 0,
				},
			},
			Outputs: []nxtblock.TOutput{
				{
					Index:        0,
					ReceiverAddr: "A",
					Amount:       10000000000000,
				},
				{
					Index:        1,
					ReceiverAddr: "B",
					Amount:       10000000000000,
				},
			},
		},
	}
	start := time.Now()
	block, err := nxtblock.NewBlock(transactions, ruleset, "0xMINER", "NXT", "DATA", lblock)
	if err != nil {
		fmt.Println("Error creating block")
		return
	}
	elapsed := time.Since(start)
	fmt.Printf("\n-- Done! (%s) - %s\n", elapsed, block.Hash)
	fmt.Println("Saved:", nxtblock.SaveBlock(*block, "blocks"))

}

func DFBC(target float64) {
	blocks, err := nxtblock.GetLatestBlocks("blocks", 10)
	if err != nil {
		fmt.Println("Error getting latest blocks")
		return
	}
	blockPtrs := make([]*nxtblock.Block, len(blocks))
	for i := range blocks {
		blockPtrs[i] = &blocks[i]
	}

	avgTime := CheckBlockTimestampForDifficulty(blockPtrs)
	if avgTime == 0 {
		fmt.Println("Not enough blocks to calculate difficulty")
		return
	}
	avgTime = math.Round(avgTime)
	fmt.Println("Average time: ", avgTime)
	fmt.Println("Target time: ", target)
	valid := avgTime < target //! 10 minutes is the target => if higher, difficulty decreases
	fmt.Println("Need to change? ", valid)
	direction := "increase"
	if avgTime > target+1 {
		direction = "decrease"
	} else if math.Abs(avgTime-target) <= 1 {
		direction = "TARGET"
	}
	fmt.Printf("Difficulty should %s\n", direction)
}

func CheckBlockTimestampForDifficulty(blocks []*nxtblock.Block) float64 {
	if len(blocks) < 2 {
		return 0
	}

	var totalTimeDiff float64
	var timeDiffs []float64

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Timestamp < blocks[j].Timestamp
	})

	for i := 1; i < len(blocks); i++ {
		timeDiff := float64(blocks[i].Timestamp-blocks[i-1].Timestamp) / 60
		totalTimeDiff += timeDiff
		timeDiffs = append(timeDiffs, timeDiff)

		fmt.Printf("Block %d - %s\n", i, time.Unix(blocks[i].Timestamp, 0).Format("15:04:05"))
		fmt.Printf("\t~ %.0f min\n", timeDiff)
	}

	avgTimeDiff := totalTimeDiff / float64(len(blocks)-1)

	fmt.Println("DIFF:", timeDiffs)
	fmt.Printf("\tDURCHSCHNITT: %.2f min\n", avgTimeDiff)

	return avgTimeDiff
}
