package id

import (
	"crypto/rand"
	"encoding/hex"
)

func New() string {
	bytes := make([]byte, 24)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
