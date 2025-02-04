package nxtblock

import (
	"crypto/sha256"
	"fmt"
	"nxtchain/nxtutxodb"
	"nxtchain/pqckpg_api"
)

type TInput struct {
	Txid      string
	Index     int    // Index of the output in the previous transaction
	Signature []byte // Signature of the transaction hash and sender's private key
	PublicKey []byte // Senders public key (base64)
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
	// Signature string // base64 encoded signature of the transaction hash and senders private key
}

// * VALIDATE TRANSACTION * //

func ValidateTransaction(transaction Transaction) (bool, error) {
	for _, input := range transaction.Inputs {
		publicKey := input.PublicKey
		signature := input.Signature
		isValid := pqckpg_api.Verify([]byte(publicKey), []byte(transaction.Hash), []byte(signature))
		if !isValid {
			return false, fmt.Errorf("invalid signature for input with txid: %s.",
				input.Txid)
		}
	}
	return true, nil
}

// * CHECK OUTPUTS AND INPUTS * //

func CheckOutputInputs(transaction Transaction) bool {
	var totalInputs int64 = 0
	var totalOutputs int64 = 0

	for _, input := range transaction.Inputs {
		amount, err := nxtutxodb.GetUTXOAmount(input.Txid, input.Index)
		if err != nil {
			return false
		}
		totalInputs += amount
	}

	for _, output := range transaction.Outputs {
		totalOutputs += output.Amount
	}

	return totalInputs >= totalOutputs
}

// * CHECK ALL UTXO FROM ONE TRANSACTION* //

func CheckTransactionUTXOs(transaction Transaction) bool {
	for _, input := range transaction.Inputs {
		_, err := nxtutxodb.GetUTXOAmount(input.Txid, input.Index)
		if err != nil {
			return false
		}
	}
	return true
}

// * CREATE TRANSACTION HEADER * //

func CreateTransactionHeader(minerAddr string, reward int64) Transaction {
	return Transaction{
		ID:     "-1",
		Hash:   fmt.Sprintf("%x", sha256.Sum256([]byte(minerAddr))),
		Inputs: []TInput{},
		Outputs: []TOutput{
			{
				Index:        0,
				ReceiverAddr: minerAddr,
				Amount:       reward,
			},
		},
	}
}

// * DELETE ALL UTXO FROM TRANSACTION * //

func DeleteTransactionUTXOs(transaction Transaction) {
	for _, input := range transaction.Inputs {
		nxtutxodb.RemoveUTXO(input.Txid, input.Index)
	}
}

// * DELETE ALL UTXO FROM BLOCK * //

func DeleteBlockUTXOs(transactions []Transaction) {
	for _, transaction := range transactions {
		DeleteTransactionUTXOs(transaction)
	}
}

// * CONVERT BLOCK TRANSACTIONS TO UTXO * //
func ConvertBlockToUTXO(block Block) {
	for _, transaction := range block.Transactions {
		for _, output := range transaction.Outputs {
			if !UTXOExists(transaction.ID, output.Index) {
				newUtxo := nxtutxodb.UTXO{
					Txid:              transaction.ID,
					Index:             output.Index,
					Amount:            output.Amount,
					PubKey:            output.ReceiverAddr,
					BlockHeight:       block.BlockHeight,
					IsHeadTransaction: false,
				}
				nxtutxodb.AddUTXOObject(newUtxo)
			}
		}
	}
	for _, transaction := range block.HeadTransactions {
		for _, output := range transaction.Outputs {
			if !UTXOExists(transaction.ID, output.Index) {
				newUtxo := nxtutxodb.UTXO{
					Txid:              transaction.ID,
					Index:             output.Index,
					Amount:            output.Amount,
					PubKey:            output.ReceiverAddr,
					BlockHeight:       block.BlockHeight,
					IsHeadTransaction: true,
				}
				nxtutxodb.AddUTXOObject(newUtxo)
			}
		}
	}
}

// * EXIST CHECK * //

func UTXOExists(txid string, index int) bool {
	for _, utxo := range nxtutxodb.UTXODatabase {
		if utxo.Txid == txid && utxo.Index == index {
			return true
		}
	}
	return false
}
