package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/omnihance/omnihance-a3-agent/internal/constants"
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

const (
	gameClientMultipartMemoryLimit = 32 * 1024 * 1024
	gameClientMultipartOverhead    = 1024 * 1024
)

func uploadFileTooLargeMessage(fileName string, maxSizeBytes int64) string {
	trimmedFileName := strings.TrimSpace(fileName)
	if trimmedFileName == "" {
		trimmedFileName = "Selected file"
	}

	return fmt.Sprintf("%s exceeds the maximum upload size of %s.", trimmedFileName, formatUploadSize(maxSizeBytes))
}

func writeGameDataUploadTooLargeError(w http.ResponseWriter, fileName string, maxSizeBytes int64) {
	_ = utils.WriteJSONResponseWithStatus(w, http.StatusRequestEntityTooLarge, map[string]interface{}{
		"errorCode": constants.ErrorCodeFileTooLarge,
		"context":   "game-data",
		"errors":    []string{uploadFileTooLargeMessage(fileName, maxSizeBytes)},
	})
}

func formatUploadSize(bytes int64) string {
	if bytes < 0 {
		bytes = 0
	}

	if bytes < 1024 {
		return fmt.Sprintf("%d Bytes", bytes)
	}

	units := []string{"KB", "MB", "GB", "TB"}
	divisor := int64(1024)
	unitIndex := 0
	for bytes/divisor >= 1024 && unitIndex < len(units)-1 {
		divisor *= 1024
		unitIndex++
	}

	value := float64(bytes) / float64(divisor)
	formatted := strconv.FormatFloat(value, 'f', 2, 64)
	formatted = strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
	return formatted + " " + units[unitIndex]
}

func gameClientMultipartMemoryBytes(maxSizeBytes int64) int64 {
	if maxSizeBytes < gameClientMultipartMemoryLimit {
		return maxSizeBytes
	}

	return gameClientMultipartMemoryLimit
}
