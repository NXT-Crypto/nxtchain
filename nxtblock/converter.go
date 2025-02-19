package nxtblock

import (
	"fmt"
	"math"
)

const NXTDivisor int64 = 100000000000
const MinValidAmount float64 = 0.00000000001

func ConvertAmount(amount int64) float64 {
	if amount > math.MaxInt64 || amount < math.MinInt64 {
		fmt.Println("Error: Amount is too large or too small for int64.")
		return 0
	}
	converted := float64(amount) / float64(NXTDivisor)

	if converted < MinValidAmount {
		return 0
	}
	return converted
}

func ConvertAmountBack(amount float64) int64 {
	result := amount * float64(NXTDivisor)
	if result > float64(math.MaxInt64) || result < float64(math.MinInt64) {
		fmt.Println("Error: Amount is too large to be converted back to int64!")
		return 0
	}
	return int64(result)
}
