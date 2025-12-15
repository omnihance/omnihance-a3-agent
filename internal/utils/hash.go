package utils

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

func GenerateMD5Hash(str string) string {
	hash := md5.Sum([]byte(strings.ToLower(str)))
	return hex.EncodeToString(hash[:])
}

func CalculateFileHash(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}
