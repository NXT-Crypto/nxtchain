package encryption

import "encoding/base64"

func Encrypt(text string) string {
	return base64.StdEncoding.EncodeToString([]byte(text))
}

func Decrypt(text string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
