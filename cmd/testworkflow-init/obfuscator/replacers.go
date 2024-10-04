package obfuscator

func FullReplace(value string) func([]byte) []byte {
	replacement := []byte(value)
	return func(_ []byte) []byte {
		return replacement
	}
}

func ShowLastCharacters(prefix string, visibleChars int) func([]byte) []byte {
	replacement := []byte(prefix)
	return func(v []byte) []byte {
		if len(v) <= visibleChars {
			return v
		}
		return append(replacement, v[len(v)-visibleChars:]...)
	}
}
