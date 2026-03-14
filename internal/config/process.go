package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const defaultStopMethodDelay = 1500 * time.Millisecond

// PriorityClass mirrors the Win32 process priority classes accepted by NSSM.
type PriorityClass string

const (
	PriorityRealtime    PriorityClass = "REALTIME_PRIORITY_CLASS"
	PriorityHigh        PriorityClass = "HIGH_PRIORITY_CLASS"
	PriorityAboveNormal PriorityClass = "ABOVE_NORMAL_PRIORITY_CLASS"
	PriorityNormal      PriorityClass = "NORMAL_PRIORITY_CLASS"
	PriorityBelowNormal PriorityClass = "BELOW_NORMAL_PRIORITY_CLASS"
	PriorityIdle        PriorityClass = "IDLE_PRIORITY_CLASS"
)

const (
	priorityRealtimeValue    uint32 = 0x00000100
	priorityHighValue        uint32 = 0x00000080
	priorityAboveNormalValue uint32 = 0x00008000
	priorityNormalValue      uint32 = 0x00000020
	priorityBelowNormalValue uint32 = 0x00004000
	priorityIdleValue        uint32 = 0x00000040
)

// AffinityMask restricts the managed process to a subset of CPUs.
type AffinityMask uint64

// StopMethodSkip is the legacy AppStopMethodSkip bitmask.
type StopMethodSkip uint32

const (
	StopMethodConsole StopMethodSkip = 1 << iota
	StopMethodWindow
	StopMethodThreads
	StopMethodTerminate
)

// StopMethodAll includes every supported stop method bit.
const StopMethodAll = StopMethodConsole | StopMethodWindow | StopMethodThreads | StopMethodTerminate

// WindowsValue converts a priority class into the DWORD stored by NSSM.
func (p PriorityClass) WindowsValue() uint32 {
	switch p {
	case PriorityRealtime:
		return priorityRealtimeValue
	case PriorityHigh:
		return priorityHighValue
	case PriorityAboveNormal:
		return priorityAboveNormalValue
	case PriorityBelowNormal:
		return priorityBelowNormalValue
	case PriorityIdle:
		return priorityIdleValue
	default:
		return priorityNormalValue
	}
}

// PriorityClassFromWindowsValue resolves a stored priority DWORD.
func PriorityClassFromWindowsValue(value uint32) (PriorityClass, bool) {
	switch value {
	case priorityRealtimeValue:
		return PriorityRealtime, true
	case priorityHighValue:
		return PriorityHigh, true
	case priorityAboveNormalValue:
		return PriorityAboveNormal, true
	case priorityNormalValue:
		return PriorityNormal, true
	case priorityBelowNormalValue:
		return PriorityBelowNormal, true
	case priorityIdleValue:
		return PriorityIdle, true
	default:
		return "", false
	}
}

// ParsePriorityClass parses CLI input for AppPriority.
func ParsePriorityClass(raw string) (PriorityClass, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "realtime_priority_class", "realtime":
		return PriorityRealtime, nil
	case "high_priority_class", "high":
		return PriorityHigh, nil
	case "above_normal_priority_class", "above-normal", "above_normal":
		return PriorityAboveNormal, nil
	case "normal_priority_class", "normal":
		return PriorityNormal, nil
	case "below_normal_priority_class", "below-normal", "below_normal":
		return PriorityBelowNormal, nil
	case "idle_priority_class", "idle":
		return PriorityIdle, nil
	default:
		return "", fmt.Errorf("invalid priority class %q", raw)
	}
}

// ParseAffinityMask parses the classic NSSM CPU list syntax used by AppAffinity.
func ParseAffinityMask(raw string) (AffinityMask, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}

	var mask AffinityMask
	for _, token := range strings.Split(raw, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			return 0, fmt.Errorf("invalid affinity mask %q", raw)
		}

		startToken, endToken, hasRange := strings.Cut(token, "-")
		if hasRange && strings.Contains(endToken, "-") {
			return 0, fmt.Errorf("invalid affinity range %q", token)
		}

		start, err := parseAffinityCPU(startToken)
		if err != nil {
			return 0, err
		}
		end := start
		if hasRange {
			end, err = parseAffinityCPU(endToken)
			if err != nil {
				return 0, err
			}
			if end < start {
				return 0, fmt.Errorf("invalid affinity range %q", token)
			}
		}

		for cpu := start; cpu <= end; cpu++ {
			mask |= AffinityMask(1) << cpu
		}
	}

	return mask, nil
}

// FormatAffinityMask renders an affinity mask using NSSM's CPU list syntax.
func FormatAffinityMask(mask AffinityMask) string {
	if mask == 0 {
		return ""
	}

	parts := make([]string, 0, 8)
	for cpu := 0; cpu < 64; {
		if mask&(AffinityMask(1)<<cpu) == 0 {
			cpu++
			continue
		}

		start := cpu
		end := cpu
		for end+1 < 64 && mask&(AffinityMask(1)<<(end+1)) != 0 {
			end++
		}

		switch end - start {
		case 0:
			parts = append(parts, strconv.Itoa(start))
		case 1:
			parts = append(parts, strconv.Itoa(start), strconv.Itoa(end))
		default:
			parts = append(parts, fmt.Sprintf("%d-%d", start, end))
		}

		cpu = end + 1
	}

	return strings.Join(parts, ",")
}

// EnabledStopMethods resolves the legacy bitmask into active stop phases.
func (s Service) EnabledStopMethods() StopMethodSkip {
	return StopMethodAll &^ s.StopMethodSkip
}

// StopWaitHint returns the total graceful-stop wait budget.
func (s Service) StopWaitHint() time.Duration {
	var total time.Duration
	methods := s.EnabledStopMethods()
	if methods&StopMethodConsole != 0 {
		total += s.StopConsoleDelay
	}
	if methods&StopMethodWindow != 0 {
		total += s.StopWindowDelay
	}
	if methods&StopMethodThreads != 0 {
		total += s.StopThreadsDelay
	}
	return total
}

func validPriorityClass(priority PriorityClass) bool {
	switch priority {
	case PriorityRealtime, PriorityHigh, PriorityAboveNormal, PriorityNormal, PriorityBelowNormal, PriorityIdle:
		return true
	default:
		return false
	}
}

func parseAffinityCPU(raw string) (uint, error) {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 6)
	if err != nil {
		return 0, fmt.Errorf("invalid affinity cpu %q", raw)
	}
	if value >= 64 {
		return 0, fmt.Errorf("affinity cpu %d is out of range", value)
	}
	return uint(value), nil
}
