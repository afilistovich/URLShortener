package random

import (
	"crypto/rand"
	"math/big"
)

const (
	idAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func GenerateString(idLength int) (string, error) {
	b := make([]byte, idLength)
	n := big.NewInt(int64(len(idAlphabet)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", err
		}
		b[i] = idAlphabet[idx.Int64()]
	}
	return string(b), nil
}
