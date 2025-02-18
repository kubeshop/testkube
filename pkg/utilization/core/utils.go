package core

import (
	"bytes"
	"fmt"
	"os"
	"runtime"

	"github.com/pkg/errors"
)

const initReadSize = 2 << 4

func readLastLine(f *os.File) (data []byte, offset int, err error) {
	var info os.FileInfo
	info, err = f.Stat()
	if err != nil {
		return
	}

	if info.IsDir() {
		err = errors.Errorf("invalid file, name:%s", f.Name())
		return
	}

	return read(f, info.Size())
}

func read(f *os.File, fileSize int64) (data []byte, n int, err error) {
	var piecesLengthArray []int64
	var buf = bytes.Buffer{}

	var sepIndex int
	var offset, sizeWillRead, sizeHasRead int64
	var b []byte
	var sep = getLineBreak()
	sizeWillRead = initReadSize

	if fileSize < int64(len(sep)) {
		data = make([]byte, fileSize)
		_, err = f.ReadAt(b, 0)
		return
	}

	// ignore the last line break if exists
	b = make([]byte, len(sep))
	if _, err = f.ReadAt(b, fileSize-int64(len(sep))); err != nil {
		return nil, 0, err
	} else if bytes.Equal(b, sep) {
		fileSize -= int64(len(sep))
	}

	for {
		sizeWillRead = sizeWillRead << 1
		offset = fileSize - (sizeWillRead + sizeHasRead)
		if offset < 0 {
			sizeWillRead = fileSize - sizeHasRead
			offset = 0
		}

		sizeHasRead += sizeWillRead
		b = make([]byte, sizeWillRead)
		_, err = f.ReadAt(b, offset)
		if err != nil {
			return
		}

		sepIndex = findSep(b, sep)
		if sepIndex == -1 {
			piecesLengthArray = append(piecesLengthArray, sizeWillRead)
			buf.Write(b)
			if sizeHasRead >= fileSize {
				return bytesReassemble(buf.Bytes(), piecesLengthArray), int(offset), nil
			}
			continue
		}
		piecesLengthArray = append(piecesLengthArray, int64(len(b)-sepIndex-len(sep)))
		buf.Write(b[sepIndex+len(sep):])

		return bytesReassemble(buf.Bytes(), piecesLengthArray), (int(offset) + sepIndex + len(sep)), nil
	}
}

func findSep(s, sep []byte) int {
	return bytes.LastIndex(s, sep)
}

func getLineBreak() []byte {
	switch runtime.GOOS {
	case "windows":
		return []byte("\r\n")
	default:
		return []byte("\n")
	}
}

// reassemble bytes array, like from [e,d,bc,a] to [a,bc,d,e]
func bytesReassemble(b []byte, piecesLengthArray []int64) (data []byte) {
	var length = int64(len(piecesLengthArray))
	var bytLength = int64(len(b))
	var sum int64
	for _, piecesLength := range piecesLengthArray {
		sum += piecesLength
	}

	if sum != bytLength {
		panic(fmt.Errorf("byes sum is not matched with sum of piecesLengthArray, %d != %d", bytLength, sum))
	}
	data = make([]byte, bytLength)
	var oldStart, start, subLength int64

	for i := length - 1; i >= 0; i-- {
		subLength = piecesLengthArray[i]
		oldStart = bytLength - subLength
		copy(data[start:start+subLength], b[oldStart:oldStart+subLength])
		start += subLength
		bytLength -= subLength
	}

	return data
}
