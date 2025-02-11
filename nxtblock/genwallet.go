package nxtblock

import (
	"crypto/sha256"
	"fmt"
	"nxtchain/pqckpg_api"

	"golang.org/x/crypto/ripemd160"
)

func GenerateWalletAddress(publicKey []byte) string {
	shaHash := sha256.Sum256(publicKey)

	ripemdHasher := ripemd160.New()
	ripemdHasher.Write(shaHash[:])
	ripemdHash := ripemdHasher.Sum(nil)

	return fmt.Sprintf("%x", ripemdHash)
}

func CreateWallet(seed []byte) Wallet {
	pk, sk := pqckpg_api.GenerateKeys(seed)
	return Wallet{
		PublicKey:  pk,
		PrivateKey: sk,
	}
}
