package core

import "strings"

// split splits a string into tokens, respecting quoted fields.
func split(s string) []string {
	var (
		tokens   []string
		sb       strings.Builder
		inQuotes bool
	)
	for i := 0; i < len(s); i++ {
		ch := s[i]

		switch ch {
		case ' ':
			if inQuotes {
				// If we're inside quotes, space is just another character
				sb.WriteByte(ch)
			} else {
				// End of a token
				if sb.Len() > 0 {
					tokens = append(tokens, sb.String())
					sb.Reset()
				}
			}
		case '"':
			// Flip inQuotes
			inQuotes = !inQuotes
			sb.WriteByte(ch)
		default:
			sb.WriteByte(ch)
		}
	}
	// If there's any leftover text in sb, push it as a token
	if sb.Len() > 0 {
		tokens = append(tokens, sb.String())
	}

	return tokens
}
