package nxtblock

import (
	"nxtchain/pqckpg_api"

	"github.com/mr-tron/base58"
	"golang.org/x/crypto/blake2b"
)

func GenerateWalletAddress(publicKey []byte) string {
	hash := blake2b.Sum256(publicKey)
	addressBytes := hash[:32]
	address := base58.Encode(addressBytes)
	return address
}

func CreateWallet(seed []byte) Wallet {
	pk, sk := pqckpg_api.GenerateKeys(seed)
	return Wallet{
		PublicKey:  pk,
		PrivateKey: sk,
	}
}
