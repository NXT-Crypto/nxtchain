package nxtblock

import (
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/ripemd160"
)

func GenerateWalletAddress(publicKey []byte) string {
	shaHash := sha256.Sum256(publicKey)

	ripemdHasher := ripemd160.New()
	ripemdHasher.Write(shaHash[:])
	ripemdHash := ripemdHasher.Sum(nil)

	return fmt.Sprintf("%x", ripemdHash)
}
