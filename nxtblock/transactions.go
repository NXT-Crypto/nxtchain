package nxtblock

type TInput struct {
	txid      string
	index     int
	signature string
	publicKey string
}

type TOutput struct {
	index        int
	amount       int64
	receiverAddr string
}

type Transaction struct {
	id        string
	timestamp int64
	hash      string
	inputs    []TInput
	outputs   []TOutput
	fee       int64
	signature string
}
