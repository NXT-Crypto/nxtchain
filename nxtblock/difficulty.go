package nxtblock

import (
	"sort"
)

func CheckBlockTimestampForDifficulty(blocks []*Block) float64 {
	if len(blocks) < 2 {
		return 0
	}

	var totalTimeDiff float64

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Timestamp < blocks[j].Timestamp
	})

	for i := 1; i < len(blocks); i++ {
		timeDiff := float64(blocks[i].Timestamp-blocks[i-1].Timestamp) / 60
		totalTimeDiff += timeDiff
	}
	return totalTimeDiff / float64(len(blocks)-1)
}
