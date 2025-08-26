package util

import (
	"crypto/rand"
	"math/big"
)

func GenerateCode() string {
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

	b := make([]rune, 6)

	for i := range b {
		rn, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[rn.Int64()]
	}

	return string(b)
}
