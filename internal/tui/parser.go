package tui

import "strings"

// parseCommandArgs parses command line arguments with proper quote handling
// Supports double quotes and escape sequences
func parseCommandArgs(input string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for _, char := range input {
		switch {
		case escaped:
			current.WriteRune(char)
			escaped = false
		case char == '\\':
			escaped = true
		case char == '"':
			inQuotes = !inQuotes
		case char == ' ' && !inQuotes:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}

	// Add the last argument if any
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}
