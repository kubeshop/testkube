package ui

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"unicode"
)

// Conofirm prints string requesting a boolean y/n confirmation
func (ui *UI) Confirm(message string) bool {
	for {
		fmt.Printf("%s [y/n] ", message)
		input, err := readInput(os.Stdin)
		if err != nil {
			log.Fatalln(err)
		}
		if r, err := parseInput(input); err == nil {
			fmt.Println()
			return r
		}
	}
}

func readInput(r io.Reader) (rune, error) {
	var buf [1]byte
	_, err := r.Read(buf[:])
	if err != nil {
		return 0, err
	}
	return rune(buf[0]), nil
}

// parseInput the rune as a bool.
// Characters y or Y return true, n or N return false.
// Everything else returns an error.
func parseInput(r rune) (bool, error) {
	switch unicode.ToLower(r) {
	case rune('y'):
		return true, nil
	case rune('n'):
		return false, nil
	}
	return false, errors.New("invalid character: " + string(r))
}
