package nxtutxodb

import (
	"errors"
	"fmt"
	"sync"
)

type UTXO struct {
	Txid              string // ID der Transaktion
	Index             int    // Index des UTXO in der Transaktion
	Amount            int64  // Amount der Transaktion
	PubKey            string // Signatur
	BlockHeight       int    // Blockhöhe, wenn größer als 100 neue blöcke, ist der UTXO erst gültig
	IsHeadTransaction bool   //schaut ob die Transaktion die Miner-Transaktion ist
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

	return utxo.Amount, nil
}

// * GET UTXODATABASE * //

func GetUTXODatabase() map[string]UTXO {
	utxoMutex.Lock()
	defer utxoMutex.Unlock()

	return UTXODatabase
}

// * SET UTXODATABASE * //

func SetUTXODatabase(utxos map[string]UTXO) {
	utxoMutex.Lock()
	defer utxoMutex.Unlock()

	UTXODatabase = utxos
}

// * ADD UTXO TO DATABASE * //

func AddUTXO(txid string, index int, amount int64, pubKey string, blockHeight int, isHeadTransaction bool) {
	key := fmt.Sprintf("%s:%d", txid, index)

	utxoMutex.Lock()
	defer utxoMutex.Unlock()

	UTXODatabase[key] = UTXO{
		Txid:              txid,
		Index:             index,
		Amount:            amount,
		PubKey:            pubKey,
		BlockHeight:       blockHeight,
		IsHeadTransaction: isHeadTransaction,
	}
}

func AddUTXOObject(utxo UTXO) {
	key := fmt.Sprintf("%s:%d", utxo.Txid, utxo.Index)

	utxoMutex.Lock()
	defer utxoMutex.Unlock()

	UTXODatabase[key] = utxo
}

// * REMOVE UTXO FROM DATABASE * //

func RemoveUTXO(txid string, index int) {
	key := fmt.Sprintf("%s:%d", txid, index)

	utxoMutex.Lock()
	defer utxoMutex.Unlock()

	delete(UTXODatabase, key)
}

// * GET UTXO FROM DATABASE BY WALLET ADDR * //

func GetUTXOByWalletAddr(walletAddr string) []UTXO {
	utxoMutex.Lock()
	defer utxoMutex.Unlock()

	var utxos []UTXO
	for _, utxo := range UTXODatabase {
		if utxo.PubKey == walletAddr {
			utxos = append(utxos, utxo)
		}
	}

	return utxos
}
