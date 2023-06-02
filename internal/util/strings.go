package util

import (
	//"crypto/rand"
	"math/rand"
)

//func GetRandomString(n int) string {
//	// set seed
//	const charset = "abcdefghijklmnopqrstuvwxyz" +
//		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
//
//	b := make([]byte, n)
//	for i := range b {
//		randIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
//		b[i] = charset[randIndex.Int64()]
//	}
//	return string(b)
//}

func GetRandomString(rng *rand.Rand, n int) string {
	// set seed
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, n)
	for i := range b {
		//randIndex, _ := rng.Int(rand.Reader, big.NewInt(int64(len(charset))))
		//b[i] = charset[randIndex.Int64()]
		b[i] = charset[rng.Intn(len(charset))]
	}
	return string(b)
}
