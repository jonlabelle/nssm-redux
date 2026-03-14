package config

import (
	"fmt"
	"strings"
)

// Hook identifies a configured AppEvents entry using NSSM's Event/Action syntax.
type Hook string

const (
	HookStartPre    Hook = "Start/Pre"
	HookStartPost   Hook = "Start/Post"
	HookStopPre     Hook = "Stop/Pre"
	HookExitPost    Hook = "Exit/Post"
	HookPowerChange Hook = "Power/Change"
	HookPowerResume Hook = "Power/Resume"
	HookRotatePre   Hook = "Rotate/Pre"
	HookRotatePost  Hook = "Rotate/Post"
)

// HookInfo exposes the event/action split used by AppEvents.
type HookInfo struct {
	Event  string
	Action string
}

// ParseHook validates and canonicalizes an AppEvents key.
func ParseHook(raw string) (Hook, error) {
	event, action, ok := strings.Cut(strings.TrimSpace(raw), "/")
	if !ok {
		return "", fmt.Errorf("invalid hook name %q", raw)
	}

	switch {
	case strings.EqualFold(event, "Start") && strings.EqualFold(action, "Pre"):
		return HookStartPre, nil
	case strings.EqualFold(event, "Start") && strings.EqualFold(action, "Post"):
		return HookStartPost, nil
	case strings.EqualFold(event, "Stop") && strings.EqualFold(action, "Pre"):
		return HookStopPre, nil
	case strings.EqualFold(event, "Exit") && strings.EqualFold(action, "Post"):
		return HookExitPost, nil
	case strings.EqualFold(event, "Power") && strings.EqualFold(action, "Change"):
		return HookPowerChange, nil
	case strings.EqualFold(event, "Power") && strings.EqualFold(action, "Resume"):
		return HookPowerResume, nil
	case strings.EqualFold(event, "Rotate") && strings.EqualFold(action, "Pre"):
		return HookRotatePre, nil
	case strings.EqualFold(event, "Rotate") && strings.EqualFold(action, "Post"):
		return HookRotatePost, nil
	default:
		return "", fmt.Errorf("invalid hook name %q", raw)
	}
}

// Info splits a canonical Hook value into its event and action names.
func (h Hook) Info() HookInfo {
	event, action, _ := strings.Cut(string(h), "/")
	return HookInfo{Event: event, Action: action}
}

// SupportedHooks returns hooks in dump order.
func SupportedHooks() []Hook {
	return []Hook{
		HookStartPre,
		HookStartPost,
		HookStopPre,
		HookExitPost,
		HookPowerChange,
		HookPowerResume,
		HookRotatePre,
		HookRotatePost,
	}
}
