package nxtblock

import (
	"encoding/json"
	"log"
	"nxtchain/nextutils"
	"os"
	"path/filepath"
	"strings"
)

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
