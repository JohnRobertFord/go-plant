package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func SignString(input string, key string) (string, error) {
	h := hmac.New(sha256.New, []byte(key))
	_, err := h.Write([]byte(input))
	if err != nil {
		return "", err
	}
	hash := h.Sum(nil)
	return hex.EncodeToString(hash), nil
}

func IsValid(msg string, mac1 string, keyString string) bool {

	mac2, err := SignString(msg, keyString)
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(mac1), []byte(mac2))
}
