package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

func SignCookie(value string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	signature := base64.URLEncoding.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%s.%s", value, signature)
}

func VerifyCookie(signedValue string, secret string) (string, error) {
	parts := strings.SplitN(signedValue, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid cookie format")
	}

	value := parts[0]
	receivedSignature := parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	expectedSignature := base64.URLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(receivedSignature), []byte(expectedSignature)) {
		return "", fmt.Errorf("invalid cookie signature")
	}

	return value, nil
}
