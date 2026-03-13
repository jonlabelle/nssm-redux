package support

import "strings"

// QuoteArg quotes an argument for a Windows command line.
func QuoteArg(arg string) string {
	if arg == "" {
		return `""`
	}

	needsQuotes := false
	for _, r := range arg {
		switch r {
		case ' ', '\t', '\n', '\v', '"':
			needsQuotes = true
		}
		if needsQuotes {
			break
		}
	}
	if !needsQuotes {
		return arg
	}

	var b strings.Builder
	b.Grow(len(arg) + 8)
	b.WriteByte('"')

	backslashes := 0
	for _, r := range arg {
		switch r {
		case '\\':
			backslashes++
		case '"':
			for range backslashes*2 + 1 {
				b.WriteByte('\\')
			}
			b.WriteByte('"')
			backslashes = 0
		default:
			for range backslashes {
				b.WriteByte('\\')
			}
			backslashes = 0
			b.WriteRune(r)
		}
	}

	for range backslashes * 2 {
		b.WriteByte('\\')
	}
	b.WriteByte('"')
	return b.String()
}

// JoinCommandLine renders arguments as a Windows command line string.
func JoinCommandLine(args []string) string {
	if len(args) == 0 {
		return ""
	}

	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = QuoteArg(arg)
	}
	return strings.Join(quoted, " ")
}
