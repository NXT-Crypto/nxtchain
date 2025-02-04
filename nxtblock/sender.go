package nxtblock

import (
	"encoding/json"
	"fmt"
	"nxtchain/nxtutxodb"
)

// * PREPARE TRANSACTION SENDER * //

func PrepareTransactionSender(transaction Transaction) (string, error) {
	jsonData, err := json.Marshal(transaction)
	if err != nil {
		return "", fmt.Errorf("failed to marshal transaction: %v", err)
	}

	return string(jsonData), nil
}

// * GET TRANSACTION SENDER * //

func GetTransactionSender(transactionStr string) (Transaction, error) {
	var transaction Transaction
	err := json.Unmarshal([]byte(transactionStr), &transaction)
	if err != nil {
		return transaction, fmt.Errorf("failed to unmarshal transaction: %v", err)
	}

	return transaction, nil
}

// * PREPARE BLOCK SENDER * //

func PrepareBlockSender(block Block) (string, error) {
	jsonData, err := json.Marshal(block)
	if err != nil {
		return "", fmt.Errorf("failed to marshal block: %v", err)
	}

	return string(jsonData), nil
}

// * GET BLOCK SENDER * //

func GetBlockSender(blockStr string) (Block, error) {
	var block Block
	err := json.Unmarshal([]byte(blockStr), &block)
	if err != nil {
		return block, fmt.Errorf("failed to unmarshal block: %v", err)
	}

	return block, nil
}

// * PREPARE UTXO SENDER * //

func PrepareUTXOSender(utxo map[string]nxtutxodb.UTXO) (string, error) {
	jsonData, err := json.Marshal(utxo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal utxo: %v", err)
	}

	return string(jsonData), nil
}

// * GET UTXO SENDER * //

func GetUTXOSender(utxoStr string) (map[string]nxtutxodb.UTXO, error) {
	utxo := make(map[string]nxtutxodb.UTXO)
	err := json.Unmarshal([]byte(utxoStr), &utxo)
	if err != nil {
		return utxo, fmt.Errorf("failed to unmarshal utxo: %v", err)
	}

	return utxo, nil
}
