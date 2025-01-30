package nxtblock

import "nxtchain/pqckpg_api"

type TInput struct {
	Txid      string
	Index     int    // Index of the output in the previous transaction
	Signature string // Signature of the transaction hash and sender's private key
	PublicKey string // Senders public key (base64)
}

type TOutput struct {
	Index        int // Index of the output in the transaction
	Amount       int64
	ReceiverAddr string // Receiver public key hash (checked later by nodes if the receiver tries to spend the output)
}

type Transaction struct {
	ID        string
	Timestamp int64
	Hash      string
	Inputs    []TInput
	Outputs   []TOutput
	Signature string // base64 encoded signature of the transaction hash and senders private key
}

func ValidateTransaction(transaction Transaction) bool {
	for _, input := range transaction.Inputs {
		isValid := pqckpg_api.Verify([]byte(input.PublicKey), []byte(transaction.Hash), []byte(input.Signature))
		if !isValid {
			return false
		}
	}
	return true
}
