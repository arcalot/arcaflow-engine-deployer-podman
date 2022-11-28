package util

import (
	"crypto/rand"
	"math/big"
)

func GetRandomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, n)
	for i := range b {
		randIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[randIndex.Int64()]
	}
	return string(b)
}
