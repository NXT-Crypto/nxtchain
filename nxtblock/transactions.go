package nxtblock

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"nxtchain/nxtutxodb"
	"nxtchain/pqckpg_api"
	"time"
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
			return false, fmt.Errorf("invalid signature for input with txid: %s",
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
		ID:     GenerateTransactionID(),
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

// * GENERATE TRANSACTION ID * //

func GenerateTransactionID() string {
	timestamp := time.Now().UnixNano()
	random := rand.New(rand.NewSource(timestamp))
	id := random.Int63()
	return fmt.Sprintf("%d", id)
}

// * GENERATE TRANSACTION HASH * //

func GenerateTransactionHash(transaction Transaction) string {
	var data string
	for _, input := range transaction.Inputs {
		data += input.Txid + fmt.Sprintf("%d", input.Index)
	}
	for _, output := range transaction.Outputs {
		data += output.ReceiverAddr + fmt.Sprintf("%d", output.Amount)
	}
	hashData := data + transaction.ID
	hashData += GenerateTransactionID()
	return fmt.Sprintf("%x", sha256.Sum256([]byte(hashData)))
}

// * PREPARE TRANSACTION * //

func PrepareTransaction(outputs []TOutput) Transaction {

	var transaction Transaction
	transaction.ID = GenerateTransactionID()
	transaction.Timestamp = time.Now().UnixNano()
	transaction.Outputs = outputs
	transaction.Hash = GenerateTransactionHash(transaction)

	return transaction
}

// * CREATE TRANSACTION INPUT * //

func CreateTransactionInput(txid string, index int, privateKey []byte, publicKey []byte, hash string) TInput {
	signature := pqckpg_api.Sign(privateKey, []byte(hash))
	return TInput{
		Txid:      txid,
		Index:     index,
		Signature: signature,
		PublicKey: publicKey,
	}
}

// * CREATE TRANSACTION OUTPUT * //

func CreateTransactionOutput(index int, amount int64, receiverAddr string) TOutput {
	return TOutput{
		Index:        index,
		Amount:       amount,
		ReceiverAddr: receiverAddr,
	}
}

// * PREPARE UTXO FOR SENDIND * //

func PrepareUTXOForWalletSender(walletAddr string) (string, map[string]nxtutxodb.UTXO, error) {
	var inputs []nxtutxodb.UTXO
	utxoMap := make(map[string]nxtutxodb.UTXO)

	for _, utxo := range nxtutxodb.GetUTXODatabase() {
		if utxo.PubKey == walletAddr {
			input := nxtutxodb.UTXO{
				Txid:              utxo.Txid,
				Index:             utxo.Index,
				Amount:            utxo.Amount,
				PubKey:            utxo.PubKey,
				BlockHeight:       utxo.BlockHeight,
				IsHeadTransaction: utxo.IsHeadTransaction,
			}
			inputs = append(inputs, input)
			utxoMap[utxo.Txid] = utxo
		}
	}

	if len(inputs) == 0 {
		return "", nil, fmt.Errorf("no UTXOs found for wallet: %s", walletAddr)
	}

	jsonData, err := json.Marshal(inputs)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal inputs: %w", err)
	}

	return string(jsonData), utxoMap, nil
}

// * RETRIEVE UTXO FROM MAP * //

func RetrieveUTXOFromJSON(jsonData string) ([]nxtutxodb.UTXO, error) {
	var utxos []nxtutxodb.UTXO
	err := json.Unmarshal([]byte(jsonData), &utxos)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal UTXOs: %w", err)
	}
	return utxos, nil
}

// * RETRIEVE TRANSACTIONS FROM MAP * //

func RetrieveTransactionsFromJSON(jsonData string) ([]Transaction, error) {
	var transactions []Transaction
	if jsonData == "{}" {
		return nil, fmt.Errorf("no transactions found")
	}
	err := json.Unmarshal([]byte(jsonData), &transactions)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal transactions: %w", err)
	}
	return transactions, nil
}
