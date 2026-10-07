package local

import (
	"math/rand/v2"
	"strings"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRandomString(length int) string {
	var sb strings.Builder
	sb.Grow(length) // Optimize memory allocation

	for i := 0; i < length; i++ {
		// Securely picks a random index from the charset
		randomIndex := rand.IntN(len(charset))
		sb.WriteByte(charset[randomIndex])
	}

	return sb.String()
}
