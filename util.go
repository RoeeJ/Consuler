package morpheus

import (
	"github.com/google/uuid"
	"math/rand"
	"strings"
)

func randomId() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func randomString(l int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, l)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
