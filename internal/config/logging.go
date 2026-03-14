package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// LogRotationSettings models NSSM's output rotation controls.
type LogRotationSettings struct {
	Enabled      bool
	Online       bool
	AgeThreshold time.Duration
	SizeBytes    uint64
	RotateDelay  time.Duration
	TimestampLog bool
}

func (s LogRotationSettings) SizeLow() uint32 {
	return uint32(s.SizeBytes & math.MaxUint32)
}

func (s LogRotationSettings) SizeHigh() uint32 {
	return uint32((s.SizeBytes >> 32) & math.MaxUint32)
}

func parseRotationSeconds(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("value is required")
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.Duration(seconds) * time.Second, nil
	}
	return time.ParseDuration(raw)
}
