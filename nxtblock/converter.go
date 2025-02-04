package nxtblock

func ConvertAmount(amount int64) float64 {
	return float64(amount) / 1000000000000
}

func ConvertAmountBack(amount float64) int64 {
	return int64(amount * 1000000000000)
}
