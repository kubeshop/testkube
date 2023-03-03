package utils

import (
	"bufio"
	"crypto/rand"
	"math/big"
	"time"
)

func ContainsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func RemoveDuplicates(s []string) []string {
	m := make(map[string]struct{})
	result := []string{}

	for _, v := range s {
		if _, value := m[v]; !value {
			m[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// RoundDuration rounds duration to default value if no round passed
func RoundDuration(duration time.Duration, to ...time.Duration) time.Duration {
	roundTo := 10 * time.Millisecond
	if len(to) > 0 {
		roundTo = to[0]
	}
	return duration.Round(roundTo)
}

// ReadLongLine reads long line
func ReadLongLine(r *bufio.Reader) (line []byte, err error) {
	var buffer []byte
	var isPrefix bool

	for {
		buffer, isPrefix, err = r.ReadLine()
		line = append(line, buffer...)
		if err != nil {
			break
		}

		if !isPrefix {
			break
		}
	}

	return line, err
}

func RandAlphanum(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)
	for i := range b {
		nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic(err)
		}
		b[i] = letters[nBig.Int64()]
	}
	return string(b)
}
