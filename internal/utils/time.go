package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseTimeRangeToSeconds(timeRange string) (int64, error) {
	if timeRange == "" {
		return 0, fmt.Errorf("time range cannot be empty")
	}

	timeRange = strings.ToLower(strings.TrimSpace(timeRange))

	var multiplier int64
	var numStr string

	if strings.HasSuffix(timeRange, "h") {
		multiplier = 3600
		numStr = strings.TrimSuffix(timeRange, "h")
	} else if strings.HasSuffix(timeRange, "d") {
		multiplier = 86400
		numStr = strings.TrimSuffix(timeRange, "d")
	} else if strings.HasSuffix(timeRange, "m") {
		multiplier = 2592000
		numStr = strings.TrimSuffix(timeRange, "m")
	} else if strings.HasSuffix(timeRange, "y") {
		multiplier = 31536000
		numStr = strings.TrimSuffix(timeRange, "y")
	} else {
		multiplier = 3600
		numStr = timeRange
	}

	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid time range format: %s", timeRange)
	}

	if num <= 0 {
		return 0, fmt.Errorf("time range must be positive: %s", timeRange)
	}

	return num * multiplier, nil
}

func GetTimeRangeStartTimestamp(timeRange string) (int64, error) {
	seconds, err := ParseTimeRangeToSeconds(timeRange)
	if err != nil {
		return 0, err
	}

	now := time.Now().Unix()
	return now - seconds, nil
}

