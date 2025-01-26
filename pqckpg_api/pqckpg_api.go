package pqckpg_api

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/g-utils/crystals-go/dilithium"
	"github.com/g-utils/crystals-go/kyber"
)

var dilithiumInstance = dilithium.NewDilithium5()
var kyberInstance = kyber.NewKyber1024()

func splitKey(combinedKey []byte) ([]byte, []byte) {
	keyParts := strings.Split(string(combinedKey), "_$-$-$_")

	if len(keyParts) != 2 {
		fmt.Println("Error: Invalid key format.")
		return nil, nil
	}

	return []byte(keyParts[0]), []byte(keyParts[1])
}

func Encrypt(publicKey []byte, message string) string {
	_, kyberPK := splitKey(publicKey)

	var chunks [][]byte
	for i := 0; i < len(message); i += 32 {
		if len(message[i:]) < 32 {
			chunks = append(chunks, []byte(message[i:]+string(make([]byte, 32-len(message[i:])))))
		} else {
			chunks = append(chunks, []byte(message[i:i+32]))
		}
	}

	var ciphertext [][]byte
	for _, chunk := range chunks {
		encChunk := kyberInstance.Encrypt(kyberPK, chunk, nil)
		ciphertext = append(ciphertext, encChunk)
	}

	var out string
	for _, chunk := range ciphertext {
		out += encode(chunk) + ","
	}
	return out
}

func Decrypt(privateKey []byte, ciphertext string) string {
	_, kyberSK := splitKey(privateKey)

	var chunks [][]byte
	for _, chunk := range strings.Split(ciphertext, ",") {
		if chunk != "" && len(decode(chunk)) != 0 {
			chunks = append(chunks, decode(chunk))
		}
	}

	var plaintext []byte
	for _, chunk := range chunks {
		plaintext = append(plaintext, kyberInstance.Decrypt(kyberSK, chunk)...)
	}

	return string(plaintext)
}

func Sign(privateKey []byte, message []byte) []byte {
	dilithiumSK, _ := splitKey(privateKey)
	return dilithiumInstance.Sign(dilithiumSK, message)
}

func Verify(publicKey []byte, message []byte, signature []byte) bool {
	dilithiumPK, _ := splitKey(publicKey)
	return dilithiumInstance.Verify(dilithiumPK, message, signature)
}

func Match(publicKey []byte, privateKey []byte) bool {
	dilithiumPK, _ := splitKey(publicKey)
	dilithiumSK, _ := splitKey(privateKey)

	message := []byte("TEST-TEST-TEST")

	signature := dilithiumInstance.Sign(dilithiumSK, message)
	return dilithiumInstance.Verify(dilithiumPK, message, signature)
}

func encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func decode(data string) []byte {
	decoded, _ := base64.StdEncoding.DecodeString(data)
	return decoded
}

func GenerateKeys(seed []byte) ([]byte, []byte) {
	Kpk, Ksk := kyberInstance.PKEKeyGen(seed)

	Dpk, Dsk := dilithiumInstance.KeyGen(seed)

	pk := append(Dpk, []byte("_$-$-$_")...)
	pk = append(pk, Kpk...)

	sk := append(Dsk, []byte("_$-$-$_")...)
	sk = append(sk, Ksk...)

	return pk, sk
}
