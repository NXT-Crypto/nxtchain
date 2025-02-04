package nxtblock

import (
	"crypto/sha256"
	"fmt"
	"nxtchain/nextutils"
	"time"
)

func ValidatorValidateTransaction(transaction Transaction) (bool, error) {
	// * 1. Schauen ob die Transaktion gültig ist (Input > Output)
	if valid := CheckOutputInputs(transaction); !valid {
		return false, fmt.Errorf("total input amount is less than output amount")
	}

	// * 2. Transaktionen validieren (Public Key, Signature des Hashes)
	valid, err := ValidateTransaction(transaction)
	if err != nil {
		return false, fmt.Errorf("transaction validation error: %v", err)
	}
	if !valid {
		return false, fmt.Errorf("invalid transaction signature or public key")
	}

	// * 3. Schauen ob die UTXO noch gültig ist und in der UTXO Datenbank vorhanden ist
	if valid := CheckTransactionUTXOs(transaction); !valid {
		return false, fmt.Errorf("UTXO not found or already spent")
	}
	return true, nil
}

func IsInputAlreadyUsed(transactions []Transaction) bool {
	var usedInputs []string
	for _, tx := range transactions {
		for _, input := range tx.Inputs {
			key := fmt.Sprintf("%s:%d", input.Txid, input.Index)
			for _, used := range usedInputs {
				if key == used {
					return true
				}
			}
			usedInputs = append(usedInputs, key)
		}
	}
	return false
}

func ValidatorValidateBlock(block Block, blockdir string, ruleset RuleSet) (bool, error) {
	// ? Block ID und Hash korrekt? (Nachbilden und vergleichen)
	blockID := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d%s%v%v%v",
		block.Timestamp,
		block.PreviousHash,
		CalculateBlockFee(block.Transactions),
		block.TransactionHash,
		block.HeadTransactions[0].Hash))))
	blockHashParts := fmt.Sprintf("%s%d%s%s%s%d%s",
		blockID,
		block.Timestamp,
		block.PreviousHash,
		block.Data,
		block.TransactionHash,
		block.Nonce,
		block.Currency,
	)
	hashBytes := sha256.Sum256([]byte(blockHashParts))
	blockHash := fmt.Sprintf("%x", hashBytes)
	if block.Hash != blockHash {
		return false, fmt.Errorf("block hash mismatch: got %s, want %s", block.Hash, blockHash)
	}
	if block.Id != blockID {
		return false, fmt.Errorf("block ID mismatch: got %s, want %s", block.Id, blockID)
	}

	// ? Previous Hash korrekt? (Vorheriger Block)
	previousBlock, err := GetLatestBlock(blockdir, false)
	if err != nil {
		return false, err
	}
	nextutils.Debug("Previous Block: %v", previousBlock)
	if block.PreviousHash != previousBlock.Hash && block.PreviousHash != "GENESIS" && previousBlock.Hash != "" {
		return false, fmt.Errorf("previous hash mismatch: got %s, want %s", block.PreviousHash, previousBlock.Hash)
	}

	// ? Timestamp korrekt? (Nicht zukunft)
	if block.Timestamp > GetTimestamp() {
		return false, fmt.Errorf("block timestamp is in the future: %d", block.Timestamp)
	}

	// ? Block Height korrekt? (Vorheriger Block + 1)
	if block.BlockHeight != previousBlock.BlockHeight+1 {
		return false, fmt.Errorf("invalid block height: got %d, want %d", block.BlockHeight, previousBlock.BlockHeight+1)
	}

	// ? Transaktionhash korrekt? (Alle Transaktionen)
	transactionHash := CalculateTransactionHash(block.Transactions)
	if block.TransactionHash != transactionHash {
		return false, fmt.Errorf("transaction hash mismatch: got %s, want %s", block.TransactionHash, transactionHash)
	}

	// ? Anzahl (Nicht mehr als MaxTransactions)
	if len(block.Transactions) > block.Ruleset.MaxTransactions {
		return false, fmt.Errorf("too many transactions in block: %d > %d", len(block.Transactions), block.Ruleset.MaxTransactions)
	}

	// ? Jede Transaktion einmalig? (Double Spending)
	if IsInputAlreadyUsed(block.Transactions) {
		return false, fmt.Errorf("double spending detected")
	}

	// ? Jede Transaktion gültig? (Transaktionen validieren)
	for _, tx := range block.Transactions {
		valid, err := ValidatorValidateTransaction(tx)
		if err != nil {
			return false, err
		}
		if !valid {
			return false, fmt.Errorf("invalid transaction %s", tx.ID)
		}
	}

	// ? Blockgebühr und Belohnung korrekt? (Block Reward)
	blockFee := CalculateBlockFee(block.Transactions)
	blockReward := CalculateBlockReward(block.Ruleset.InitialReward, int64(block.BlockHeight))
	fullReward := blockFee + blockReward
	if block.HeadTransactions[0].Outputs[0].Amount != fullReward {
		return false, fmt.Errorf("invalid block reward: got %d, want %d", block.HeadTransactions[0].Outputs[0].Amount, fullReward)
	}

	// ? Sind alle UTXO Inputs gelöscht? (Lokale UTXO Datenbank checken)
	for _, tx := range block.Transactions {
		DeleteTransactionUTXOs(tx)
	}

	// ? Sind alle UTXO Outputs erstellt? (Lokale UTXO Datenbank checken)
	ConvertBlockToUTXO(block)

	// ? Ist das Ruleset gleich und die Difficulty? (Regeln)
	if block.Ruleset.Difficulty != ruleset.Difficulty {
		return false, fmt.Errorf("invalid difficulty: got %d, want %d", block.Ruleset.Difficulty, ruleset.Difficulty)
	}
	if block.Ruleset.Version != ruleset.Version {
		return false, fmt.Errorf("invalid version: got %d, want %d", block.Ruleset.Version, ruleset.Version)
	}
	if block.Ruleset.MaxTransactions != ruleset.MaxTransactions {
		return false, fmt.Errorf("invalid max transactions: got %d, want %d", block.Ruleset.MaxTransactions, ruleset.MaxTransactions)
	}

	return true, nil
}

func GetTimestamp() int64 {
	return time.Now().Unix()
}
