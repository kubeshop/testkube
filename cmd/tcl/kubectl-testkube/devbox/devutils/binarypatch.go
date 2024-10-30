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
	"math"
	"slices"
	"time"
)

const (
	BinaryPatchAddOp      = 1
	BinaryPatchOriginalOp = 2
	BinaryPatchDeleteOp   = 3
	BinaryPatchRepeatOp   = 4

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
	skew := int32(30)
	minReuse := int32(12)
	step := skew / 2

	ts := time.Now()

	originalMarkers := make([][]int32, math.MaxUint16+1)
	originalIterations := int32(len(originalFile)) - skew
	for i := skew; i < originalIterations; i++ {
		if originalFile[i] == 0 {
			continue
		}
		// Approximate the marker
		marker := uint16((int(originalFile[i-(skew/4)])+int(originalFile[i-(skew/2)]))/2) | uint16((int(originalFile[i+(skew/4)])+int(originalFile[i+(skew/2)]))/2)<<8
		originalMarkers[marker] = append(originalMarkers[marker], i)
	}

	// Delete most popular characters to avoid problems with too many iterations
	lenSum := 0
	for i := uint16(0); i < math.MaxUint16; i++ {
		lenSum += len(originalMarkers[i])
	}
	lenAvg := lenSum / len(originalMarkers)
	for i := uint16(0); i < math.MaxUint16; i++ {
		if len(originalMarkers[i]) > lenAvg {
			originalMarkers[i] = nil
		}
	}

	// Sort all the markers
	for i := uint16(0); i < math.MaxUint16; i++ {
		slices.Sort(originalMarkers[i])
	}

	ciMax := int32(len(currentFile) - 1)
	oiMax := int32(len(originalFile) - 1)

	lastOi := int32(0)
	lastCi := int32(0)
	totalSaved := 0
	iterations := 0

	maxIndex := int32(len(currentFile)) - skew
loop:
	for ci := skew; ci < maxIndex; {
		iterations++
		if currentFile[ci] == 0 {
			ci += step
			continue
		}
		marker := uint16((int(currentFile[ci-(skew/4)])+int(currentFile[ci-(skew/2)]))/2) | uint16((int(currentFile[ci+(skew/4)])+int(currentFile[ci+(skew/2)]))/2)<<8

		if maxDuration != 0 && iterations%1000 == 0 && time.Since(ts) > maxDuration {
			break
		}

		if iterations%50000 == 0 {
			step = step * 7 / 6
			continue
		}

		nextCL := 0
		nextCR := 0
		nextOL := 0
		for _, oi := range originalMarkers[marker] {
			if currentFile[ci] != originalFile[oi] ||
				currentFile[ci+1] != originalFile[oi+1] ||
				currentFile[ci-1] != originalFile[oi-1] ||
				currentFile[ci-skew/2] != originalFile[oi-skew/2] ||
				currentFile[ci+skew/2] != originalFile[oi+skew/2] {
				continue
			}
			// Validate exact range
			l, r := int32(0), int32(0)
			for ; ci-l > lastCi && oi-l > lastOi && originalFile[oi-l-1] == currentFile[ci-l-1]; l++ {
			}
			for ; oi+r < oiMax && ci+r < ciMax && originalFile[oi+r+1] == currentFile[ci+r+1]; r++ {
			}
			// Determine if it's nice
			if l+r+1 >= minReuse && nextCR-nextCL < int(r+l) {
				nextCL = int(ci - l)
				nextCR = int(ci + r)
				nextOL = int(oi - l)
			}
			if l+r > skew {
				break
			}
		}

		if nextCL != 0 || nextCR != 0 {
			totalSaved += (nextCR - nextCL) + 1
			p.Add(currentFile[lastCi:nextCL])
			lastCi = int32(nextCR + 1)
			p.Original(nextOL, (nextCR-nextCL)+1)
			ci = lastCi + 1
			continue loop
		}

		ci += step
	}

	p.Add(currentFile[lastCi:])
}

func (p *BinaryPatch) Len() int {
	return p.buf.Len()
}

func (p *BinaryPatch) Original(index, bytesCount int) {
	if bytesCount == 0 {
		return
	}
	p.lastOp = BinaryPatchOriginalOp
	p.buf.WriteByte(BinaryPatchOriginalOp)
	num := make([]byte, 4)
	binary.LittleEndian.PutUint32(num, uint32(index))
	p.buf.Write(num)
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

func (p *BinaryPatch) Repeat(count int, b byte) {
	if count == 0 {
		return
	}
	p.lastOp = BinaryPatchRepeatOp
	p.buf.WriteByte(BinaryPatchRepeatOp)
	num := make([]byte, 4)
	binary.LittleEndian.PutUint32(num, uint32(count))
	p.buf.Write(num)
	p.buf.WriteByte(b)
}

func (p *BinaryPatch) Apply(original []byte) []byte {
	result := make([]byte, 0)
	patch := p.buf.Bytes()
	for i := 0; i < len(patch); {
		switch patch[i] {
		case BinaryPatchOriginalOp:
			index := binary.LittleEndian.Uint32(patch[i+1 : i+5])
			count := binary.LittleEndian.Uint32(patch[i+5 : i+9])
			result = append(result, original[index:index+count]...)
			i += 9
		case BinaryPatchRepeatOp:
			count := binary.LittleEndian.Uint32(patch[i+1 : i+5])
			buf := make([]byte, count)
			for j := 0; j < int(count); j++ {
				buf[j] = patch[i+5]
			}
			result = append(result, buf...)
			i += 6
		case BinaryPatchAddOp:
			count := binary.LittleEndian.Uint32(patch[i+1 : i+5])
			result = append(result, patch[i+5:i+5+int(count)]...)
			i += 5 + int(count)
		}
	}
	return result
}
