// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devutils

import (
	"bytes"
	"encoding/binary"
	"time"
)

const (
	BinaryPatchAddOp      = 1
	BinaryPatchPreserveOp = 2
	BinaryPatchDeleteOp   = 3

	BinaryBatchBlockSize      = 100
	BinaryBatchBlockFactor    = 8
	BinaryBatchBlockFactorMax = 50
	BinaryPatchMinCommonSize  = 8
)

// BinaryPatch is helper to avoid sending the whole binaries.
// It's optimized for fast analysis to send it ASAP,
// so the resulting patch may be bigger than it's needed.
// It's working nicely for incremental builds though.
type BinaryPatch struct {
	buf       *bytes.Buffer
	lastOp    int
	lastCount int
}

type BinaryPatchThreshold struct {
	Duration time.Duration
	Minimum  float64
}

func NewBinaryPatch() *BinaryPatch {
	return &BinaryPatch{
		buf: bytes.NewBuffer(nil),
	}
}

func NewBinaryPatchFromBytes(data []byte) *BinaryPatch {
	return &BinaryPatch{
		buf: bytes.NewBuffer(data),
	}
}

func NewBinaryPatchFor(originalFile, currentFile []byte, maxDuration time.Duration) *BinaryPatch {
	p := NewBinaryPatch()
	p.Read(originalFile, currentFile, maxDuration)
	return p
}

func (p *BinaryPatch) Bytes() []byte {
	return p.buf.Bytes()
}

func (p *BinaryPatch) Load(data []byte) {
	p.buf = bytes.NewBuffer(data)
}

func (p *BinaryPatch) Read(originalFile, currentFile []byte, maxDuration time.Duration) {
	originalFileLen := len(originalFile)
	currentFileLen := len(currentFile)
	smallerFileLen := min(originalFileLen, currentFileLen)

	iteration := 0
	currentIndex := 0
	originalIndex := 0

	// Omit same sequence
	omit := 0
	for omit < smallerFileLen && currentFile[omit] == originalFile[omit] {
		omit++
	}
	originalIndex += omit
	currentIndex += omit
	p.Preserve(omit)

	replaced := 0
	ts := time.Now()

loop:
	for {
		iteration++
		leftCurrent := currentFileLen - currentIndex
		leftOriginal := originalFileLen - originalIndex
		leftMin := min(leftCurrent, leftOriginal)
		if leftMin <= BinaryPatchMinCommonSize {
			break
		}
		segment := min(leftMin, BinaryBatchBlockSize*BinaryBatchBlockFactor) - BinaryPatchMinCommonSize
		maxIterations := min(segment+replaced, leftMin)

		// Extract fast when duration passed
		if maxDuration != 0 && iteration%1000 == 0 && maxDuration < time.Since(ts) {
			p.Delete(originalFileLen - originalIndex)
			p.Add(currentFile[currentIndex:])
			return
		}

		// Try recovering from an endless loop
		if maxIterations > BinaryBatchBlockSize*BinaryBatchBlockFactorMax {
			p.Add(currentFile[currentIndex : currentIndex+maxIterations/2])
			replaced -= maxIterations / 2
			currentIndex += maxIterations / 2
			continue loop
		}

		// Find next match when adding to original file
		biggestCommon := 0
		biggestCommonAt := 0
		biggestCommonDel := false
		for i := 0; i < maxIterations; i++ {
			common := 0
			for i+common < leftMin && currentFile[currentIndex+i+common] == originalFile[originalIndex+common] {
				common++
			}
			if common > biggestCommon {
				biggestCommon = common
				biggestCommonAt = i
				biggestCommonDel = false
			}
			common = 0
			for i+common < leftMin && currentFile[currentIndex+common] == originalFile[originalIndex+i+common] {
				common++
			}
			if common > biggestCommon {
				biggestCommon = common
				biggestCommonAt = i
				biggestCommonDel = true
			}
		}

		if biggestCommon >= BinaryPatchMinCommonSize {
			if biggestCommonDel {
				p.Delete(biggestCommonAt)
				p.Preserve(biggestCommon)
				replaced += biggestCommonAt
				originalIndex += biggestCommonAt + biggestCommon
				currentIndex += biggestCommon
			} else {
				p.Add(currentFile[currentIndex : currentIndex+biggestCommonAt])
				p.Preserve(biggestCommon)
				replaced -= biggestCommonAt
				currentIndex += biggestCommonAt + biggestCommon
				originalIndex += biggestCommon
			}
			continue loop
		}

		// Treat some part as deleted to proceed
		p.Delete(BinaryPatchMinCommonSize)
		replaced += BinaryPatchMinCommonSize
		originalIndex += BinaryPatchMinCommonSize
	}

	if currentIndex != currentFileLen {
		p.Add(currentFile[currentIndex:])
	} else if originalIndex != originalFileLen {
		p.Preserve(originalFileLen - originalIndex)
	}
}

func (p *BinaryPatch) Len() int {
	return p.buf.Len()
}

func (p *BinaryPatch) Preserve(bytesCount int) {
	if bytesCount == 0 {
		return
	}
	if p.lastOp == BinaryPatchPreserveOp {
		p.lastCount += bytesCount
		b := p.buf.Bytes()
		binary.LittleEndian.PutUint32(b[len(b)-4:], uint32(p.lastCount))
		return
	}
	p.lastOp = BinaryPatchPreserveOp
	p.lastCount = bytesCount
	p.buf.WriteByte(BinaryPatchPreserveOp)
	num := make([]byte, 4)
	binary.LittleEndian.PutUint32(num, uint32(bytesCount))
	p.buf.Write(num)
}

func (p *BinaryPatch) Delete(bytesCount int) {
	if bytesCount == 0 {
		return
	}
	if p.lastOp == BinaryPatchDeleteOp {
		p.lastCount += bytesCount
		b := p.buf.Bytes()
		binary.LittleEndian.PutUint32(b[len(b)-4:], uint32(p.lastCount))
		return
	}
	p.lastOp = BinaryPatchDeleteOp
	p.lastCount = bytesCount
	p.buf.WriteByte(BinaryPatchDeleteOp)
	num := make([]byte, 4)
	binary.LittleEndian.PutUint32(num, uint32(bytesCount))
	p.buf.Write(num)
}

func (p *BinaryPatch) Add(bytesArr []byte) {
	if len(bytesArr) == 0 {
		return
	}
	if p.lastOp == BinaryPatchAddOp {
		b := p.buf.Bytes()
		nextCount := p.lastCount + len(bytesArr)
		binary.LittleEndian.PutUint32(b[len(b)-p.lastCount-4:], uint32(nextCount))
		p.buf.Write(bytesArr)
		p.lastCount = nextCount
		return
	}
	p.lastOp = BinaryPatchAddOp
	p.lastCount = len(bytesArr)
	p.buf.WriteByte(BinaryPatchAddOp)
	num := make([]byte, 4)
	binary.LittleEndian.PutUint32(num, uint32(len(bytesArr)))
	p.buf.Write(num)
	p.buf.Write(bytesArr)
}

func (p *BinaryPatch) Apply(original []byte) []byte {
	result := make([]byte, 0)
	patch := p.buf.Bytes()
	for i := 0; i < len(patch); {
		switch patch[i] {
		case BinaryPatchPreserveOp:
			count := binary.LittleEndian.Uint32(patch[i+1 : i+5])
			result = append(result, original[:count]...)
			original = original[count:]
			i += 5
		case BinaryPatchDeleteOp:
			count := binary.LittleEndian.Uint32(patch[i+1 : i+5])
			original = original[count:]
			i += 5
		case BinaryPatchAddOp:
			count := binary.LittleEndian.Uint32(patch[i+1 : i+5])
			result = append(result, patch[i+5:i+5+int(count)]...)
			i += 5 + int(count)
		}
	}
	return result
}
