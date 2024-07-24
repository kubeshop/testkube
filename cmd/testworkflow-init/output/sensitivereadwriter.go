package output

import (
	"io"
	"sync"
	"unsafe"
)

type sensitiveReadWriter struct {
	dst         io.Writer
	replacement []byte

	rootNode        *SearchTree
	currentNode     *SearchTree
	currentBuffer   []byte
	currentPosition int
	currentEnd      int

	writeMu sync.Mutex
}

func newSensitiveReadWriter(dst io.Writer, replacement string, words []string) *sensitiveReadWriter {
	// Build radix-tree for checking the words
	rootNode := NewSearchTree()
	for _, word := range words {
		rootNode.Append(unsafe.Slice(unsafe.StringData(word), len(word)))
	}

	return &sensitiveReadWriter{
		dst:         dst,
		replacement: []byte(replacement),
		rootNode:    rootNode,
		currentNode: rootNode,
		currentEnd:  -1,
	}
}

func (s *sensitiveReadWriter) SetSensitiveReplacement(replacement string) {
	s.replacement = []byte(replacement)
}

func (s *sensitiveReadWriter) SetSensitiveWords(words []string) {
	rootNode := NewSearchTree()
	for _, word := range words {
		rootNode.Append(unsafe.Slice(unsafe.StringData(word), len(word)))
	}

	s.rootNode = rootNode
	s.currentPosition = 0
	s.currentEnd = -1
	s.currentNode = s.rootNode
}

func (s *sensitiveReadWriter) resetBuffer() {
	s.currentBuffer = nil
	s.currentPosition = 0
	s.currentEnd = -1
	s.currentNode = s.rootNode
}

func (s *sensitiveReadWriter) Write(p []byte) (n int, err error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	// Read data from the previous cycle
	if s.currentBuffer != nil {
		p = append(s.currentBuffer, p...)
	}
	currentPosition := s.currentPosition
	currentStart := currentPosition
	currentEnd := s.currentEnd
	currentNode := s.currentNode
	s.resetBuffer()

	var nn int
	for currentPosition < len(p) {
		end, depth, mayContinue, current := currentNode.Hits(p, currentPosition)

		// Continue with next characters when there was no hit
		if end == -1 && !mayContinue {
			currentPosition++
			continue
		}

		// Flush the non-sensitive contents if there is a potential hit
		if currentPosition != currentStart {
			currentStart = 0
			nn, err = s.dst.Write(p[:currentPosition])
			n += nn
			if err != nil {
				return
			}

			// Calibrate data without the non-sensitive content
			p = p[currentPosition:]
			depth -= currentPosition
			if end != -1 {
				end -= currentPosition
			}
			currentPosition = 0
		}

		// Adjust the current end character
		if end != -1 {
			currentEnd = end
		}

		// The sensitive word may still not be finished in this buffer
		if mayContinue {
			// Buffer data
			s.currentBuffer = p
			s.currentNode = current
			s.currentPosition = depth
			s.currentEnd = currentEnd

			// End a call
			return n + len(p), nil
		}

		// Flush the acknowledged sensitive data
		nn, err = s.dst.Write(s.replacement)
		nn += currentEnd - len(s.replacement)
		n += nn
		p = p[currentEnd:]
		currentEnd = -1
		currentPosition = 0
		currentNode = s.rootNode
		if err != nil {
			return n, err
		}
	}

	// Write the rest of data
	if len(p) > 0 {
		nn, err = s.dst.Write(p)
		return n + nn, err
	}
	return n, nil
}

func (s *sensitiveReadWriter) Flush() error {
	for s.currentBuffer != nil {
		// Flush all if there is no smaller sensitive chunk
		if s.currentEnd == -1 {
			left := s.currentBuffer
			s.resetBuffer()

			_, err := s.dst.Write(left)
			if err != nil {
				return err
			}
			return nil
		}

		// Flush the next sensitive part
		_, err := s.dst.Write(s.replacement)
		if err != nil {
			return err
		}

		// Write the remaining part
		left := s.currentBuffer[s.currentEnd:]
		s.resetBuffer()
		_, err = s.Write(left)
		if err != nil {
			return err
		}
	}
	return nil
}
