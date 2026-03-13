package support

import (
	"slices"
	"strings"
)

// MergeEnvironment applies the NSSM-style environment merge rules.
func MergeEnvironment(base, override, extra []string) []string {
	var merged []string
	if len(override) > 0 {
		merged = slices.Clone(override)
	} else {
		merged = slices.Clone(base)
	}

	index := make(map[string]int, len(merged))
	for i, entry := range merged {
		if key, ok := splitEnvKey(entry); ok {
			index[strings.ToUpper(key)] = i
		}
	}

	for _, entry := range extra {
		key, ok := splitEnvKey(entry)
		if !ok {
			continue
		}

		upperKey := strings.ToUpper(key)
		if i, ok := index[upperKey]; ok {
			merged[i] = entry
			continue
		}

		index[upperKey] = len(merged)
		merged = append(merged, entry)
	}

	return merged
}

func splitEnvKey(entry string) (string, bool) {
	key, _, ok := strings.Cut(entry, "=")
	if !ok || key == "" {
		return "", false
	}
	return key, true
}
