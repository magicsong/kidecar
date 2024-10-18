package utils

import (
	"crypto/sha256"
	"fmt"
)

// Hash hashes `data` with sha256 and returns the hex string
func Hash(data string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
}