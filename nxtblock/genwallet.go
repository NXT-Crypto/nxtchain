package nxtblock

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"nxtchain/pqckpg_api"
)

func GenerateWalletAddress(publicKey []byte) string {
	pkHash := sha256.Sum256(publicKey)

	sha512Hasher := sha512.New()
	sha512Hasher.Write(pkHash[:])
	shaHash := sha512Hasher.Sum(nil)
	return fmt.Sprintf("%x", shaHash)
}

func CreateWallet(seed []byte) Wallet {
	pk, sk := pqckpg_api.GenerateKeys(seed)
	return Wallet{
		PublicKey:  pk,
		PrivateKey: sk,
	}
}
