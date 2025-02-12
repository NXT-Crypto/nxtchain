package nxtblock

import (
	"encoding/json"
	"log"
	"nxtchain/nextutils"
	"os"
	"path/filepath"
	"strings"
)

type Wallet struct {
	PrivateKey []byte `json:"private_key"`
	PublicKey  []byte `json:"public_key"`
}

// * SAVE BLOCK * //

func SaveBlock(block Block, dir string) string {
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Error creating directory: %v", err)
		return ""
	}

	blockJSON, err := json.Marshal(block)
	if err != nil {
		log.Printf("Error marshaling block: %v", err)
		return ""
	}

	path := filepath.Join(dir, block.Hash+".json")
	if err := os.WriteFile(path, blockJSON, 0644); err != nil {
		log.Printf("Error writing block file: %v", err)
		return ""
	}
	return path
}

// * LOAD BLOCK * //

func LoadBlock(hash, dir string) (Block, error) {
	var filename string = ""
	if !strings.HasSuffix(hash, ".json") {
		filename = filepath.Join(dir, hash+".json")
	} else {
		filename = filepath.Join(dir, hash)
	}

	blockJSON, err := os.ReadFile(filename)
	if err != nil {
		return Block{}, err
	}

	var block Block
	if err := json.Unmarshal(blockJSON, &block); err != nil {
		return Block{}, err
	}
	return block, nil
}

// * DELETE BLOCK * //

func DeleteBlock(hash, dir string) error {
	filename := filepath.Join(dir, hash+".json")
	return os.Remove(filename)
}

// * GET LATEST BLOCK * //

func GetLatestBlock(dir string, ignoreEmpty bool) (Block, error) {
	nextutils.Debug("Getting latest block from %s", filepath.Clean(dir))
	files, err := os.ReadDir(dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		nextutils.Error("Error creating directory: %v", err)
		return Block{}, err
	}
	if err != nil {
		nextutils.Error("Error reading directory: %v", err)
		return Block{}, err
	}
	nextutils.Debug("Found %d files", len(files))

	var latestBlock Block
	for _, file := range files {
		nextutils.Debug("Checking file %s", file.Name())
		if file.IsDir() {
			nextutils.Debug("Skipping non-directory %s", file.Name())
			continue
		}

		block, err := LoadBlock(file.Name(), dir)
		nextutils.Debug("Loaded block %s", block.Hash)
		if err != nil {
			nextutils.Error("Error loading block: %v", err)
			continue
		}

		if block.Timestamp > latestBlock.Timestamp {
			latestBlock = block
		}
	}
	if latestBlock.Hash == "" && !ignoreEmpty {
		return Block{}, os.ErrNotExist
	}
	nextutils.Debug("Latest block (hash): %v", latestBlock.Hash)
	return latestBlock, nil
}

// * GET LATEST BLOCKS * //

func GetLatestBlocks(dir string, count int) ([]Block, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var blocks []Block
	for _, file := range files {
		block, err := LoadBlock(file.Name(), dir)
		if err != nil {
			continue
		}
		blocks = append(blocks, block)
	}
	if len(blocks) == 0 {
		return nil, os.ErrNotExist
	}
	if count > len(blocks) {
		count = len(blocks)
	}
	return blocks[:count], nil
}

// * GET LOCAL BLOCKHEIGHT * //

func GetLocalBlockHeight(dir string) int {
	ltstblock, err := GetLatestBlock(dir, false)
	if err != nil {
		return 0
	}
	return ltstblock.BlockHeight
}

// * GET BLOCK BY HEIGHT * //

func GetBlockByHeight(height int, dir string) (Block, error) {
	if height < 0 {
		return Block{}, os.ErrInvalid
	}

	if height == 0 {
		return GetLatestBlock(dir, false)
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return Block{}, err
	}

	if len(files) == 0 {
		return Block{}, os.ErrNotExist
	}

	for _, file := range files {
		block, err := LoadBlock(file.Name(), dir)
		if err != nil {
			continue
		}
		if block.BlockHeight == height {
			return block, nil
		}
	}
	return Block{}, os.ErrNotExist
}

// * SAVE WALLET * //

func SaveWallet(wallet Wallet, dir string) string {
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Error creating directory: %v", err)
		return ""
	}

	walletJSON, err := json.Marshal(wallet)
	if err != nil {
		log.Printf("Error marshaling wallet: %v", err)
		return ""
	}

	path := filepath.Join(dir, GenerateWalletAddress(wallet.PublicKey)+".json")
	if err := os.WriteFile(path, walletJSON, 0644); err != nil {
		log.Printf("Error writing wallet file: %v", err)
		return ""
	}
	return path
}

// * LOAD WALLET * //

func LoadWallet(address, dir string) (Wallet, error) {
	var filename string = ""
	if !strings.HasSuffix(address, ".json") {
		filename = filepath.Join(dir, address+".json")
	} else {
		filename = filepath.Join(dir, address)
	}

	walletJSON, err := os.ReadFile(filename)
	if err != nil {
		return Wallet{}, err
	}

	var wallet Wallet
	if err := json.Unmarshal(walletJSON, &wallet); err != nil {
		return Wallet{}, err
	}
	return wallet, nil
}

// * DELETE WALLET * //

func DeleteWallet(address, dir string) error {
	filename := filepath.Join(dir, address+".json")
	return os.Remove(filename)
}

// * GET ALL WALLETS * //

func GetAllWallets(dir string) ([]Wallet, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var wallets []Wallet
	for _, file := range files {
		wallet, err := LoadWallet(file.Name(), dir)
		if err != nil {
			continue
		}
		wallets = append(wallets, wallet)
	}
	return wallets, nil
}

// * GET ALL TRANSACTIONS BY WALLET (Getting input & outputs) * //

func GetAllTransactionsFromBlocks(blockdir string, walletaddr string) map[string]Transaction {
	files, err := os.ReadDir(blockdir)
	transactions := make(map[string]Transaction)
	if err != nil {
		log.Printf("Error reading directory: %v", err)
		return transactions
	}

	var blocks []Block

	for _, file := range files {
		block, err := LoadBlock(file.Name(), blockdir)
		if err != nil {
			continue
		}
		blocks = append(blocks, block)
	}

	for _, block := range blocks {
		for _, tx := range block.Transactions {
			for _, input := range tx.Inputs {
				if string(input.PublicKey) == walletaddr {
					transactions[tx.Hash] = tx
				}
			}
			for _, output := range tx.Outputs {
				if string(output.ReceiverAddr) == walletaddr {
					transactions[tx.Hash] = tx
				}
			}
		}
	}

	return transactions
}

// * GET ALL BLOCKS * //

func GetAllBlocks(dir string) ([]Block, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var blocks []Block
	for _, file := range files {
		block, err := LoadBlock(file.Name(), dir)
		if err != nil {
			continue
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// * GET BLOCK BY HASH * //

func GetBlockByHash(dir, hash string) (Block, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return Block{}, err
	}

	for _, file := range files {
		if file.Name() == hash+".json" {
			return LoadBlock(hash, dir)
		}
	}
	return Block{}, os.ErrNotExist
}
