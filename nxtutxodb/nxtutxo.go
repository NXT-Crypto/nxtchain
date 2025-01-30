package nxtutxodb

import (
	"errors"
	"fmt"
	"sync"
)

type UTXO struct {
	txid              string // ID der Transaktion
	index             int    // Index des UTXO in der Transaktion
	amount            int64  // Amount der Transaktion
	pubKey            string // Signatur
	blockHeight       int    // Blockhöhe, wenn größer als 100 neue blöcke, ist der UTXO erst gültig
	isHeadTransaction bool   //schaut ob die Transaktion die Miner-Transaktion ist
}

var UTXODatabase = make(map[string]UTXO)
var utxoMutex sync.Mutex

// * GET UTXO FROM TRANSACTION * //

func GetUTXOAmount(txid string, index int) (int64, error) {
	key := fmt.Sprintf("%s:%d", txid, index)

	utxoMutex.Lock()
	defer utxoMutex.Unlock()

	utxo, exists := UTXODatabase[key]
	if !exists {
		return 0, errors.New("UTXO not found")
	}

	return utxo.amount, nil
}

// * ADD UTXO TO DATABASE * //

func AddUTXO(txid string, index int, amount int64, pubKey string, blockHeight int, isHeadTransaction bool) {
	key := fmt.Sprintf("%s:%d", txid, index)

	utxoMutex.Lock()
	defer utxoMutex.Unlock()

	UTXODatabase[key] = UTXO{
		txid:              txid,
		index:             index,
		amount:            amount,
		pubKey:            pubKey,
		blockHeight:       blockHeight,
		isHeadTransaction: isHeadTransaction,
	}
}

// * REMOVE UTXO FROM DATABASE * //

func RemoveUTXO(txid string, index int) {
	key := fmt.Sprintf("%s:%d", txid, index)

	utxoMutex.Lock()
	defer utxoMutex.Unlock()

	delete(UTXODatabase, key)
}
