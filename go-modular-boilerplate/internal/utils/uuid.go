package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

const (
	charset  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	idLength = 18
)

// GenerateUUID generates a new UUID string
func GenerateUUID() string {
	return uuid.New().String()
}

func getNanoIDWithSize(idSize int) string {
	const TIMESTAMP_LENGTH = 11 // length of base36 encoded timestamp in milliseconds

	if idSize <= TIMESTAMP_LENGTH {
		return uuid.New().String()[:idSize]
	}

	// Use base36 encoded timestamp for compactness
	ts := fmt.Sprintf("%x-", time.Now().UTC().UnixMilli()) // hex is shorter than decimal
	tsLen := len(ts)

	// If timestamp is longer than idSize, truncate timestamp
	var prefix string
	var randLen int
	if tsLen >= idSize {
		prefix = ts[:idSize]
		randLen = 0
	} else {
		prefix = ts
		randLen = idSize - tsLen
	}

	result := make([]byte, randLen)
	for i := 0; i < randLen; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return ""
		}
		result[i] = charset[num.Int64()]
	}

	return prefix + string(result)
}

func GetNanoIDWithPrefix(prefix string) string {
	result := getNanoIDWithSize(idLength)
	return prefix + "-" + result
}
