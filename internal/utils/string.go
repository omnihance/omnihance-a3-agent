package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strings"
)

func GenerateRandomToken(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return strings.Repeat("0", length)
	}

	return hex.EncodeToString(bytes)
}

func ReadStringFromBytes(buffer []byte) string {
	nullIndex := bytes.IndexByte(buffer, 0)
	if nullIndex == -1 {
		return string(buffer)
	}

	return string(buffer[:nullIndex])
}
