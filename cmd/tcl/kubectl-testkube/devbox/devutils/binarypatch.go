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

type BinaryPatchOpType = byte

const (
	BinaryPatchAddOpType      BinaryPatchOpType = 1
	BinaryPatchOriginalOpType BinaryPatchOpType = 2
)

// BinaryPatch is helper to avoid sending the whole binaries.
// It's optimized for fast analysis to send it ASAP,
// so the resulting patch may be bigger than it's needed.
// It's working nicely for incremental builds though.
type BinaryPatch struct {
	buf *bytes.Buffer
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
	skew := uint32(50)
	minReuse := uint32(20)
	reasonableReuse := uint32(128)
	step := skew / 2

	ops := &BinaryPatchOpList{}

	ts := time.Now()

	originalMarkers := make([][]uint32, math.MaxUint16+1)
	originalIterations := uint32(len(originalFile)) - skew

	for i := skew; i < originalIterations; {
		if originalFile[i] == 0 {
			i++
			continue
		}
		// Approximate the marker
		marker := uint16((int(originalFile[i-(skew/4)])+int(originalFile[i-(skew/2)]))/2) | uint16((int(originalFile[i+(skew/4)])+int(originalFile[i+(skew/2)]))/2)<<8
		originalMarkers[marker] = append(originalMarkers[marker], i)
		i++
	}

	// Delete most popular characters to avoid problems with too many iterations
	sizes := make([]int, len(originalMarkers))
	for i := 0; i < len(originalMarkers); i++ {
		sizes[i] = len(originalMarkers[i])
	}
	slices.Sort(sizes)
	total := 0
	for i := range originalMarkers {
		total += len(originalMarkers[i])
	}
	current := total
	clearTopMarkers := func(percentage int) {
		percentage = max(100-max(0, percentage), 0)
		keep := total * percentage / 100
		i := 0
		for ; i < len(sizes) && keep > 0; i++ {
			keep -= sizes[i]
		}
		if i == len(sizes) {
			i--
		}
		maxMarkersCount := sizes[i]
		for j := i + 1; j < len(sizes); j++ {
			sizes[j] = 0
		}

		for i := uint16(0); i < math.MaxUint16; i++ {
			if len(originalMarkers[i]) > maxMarkersCount {
				current -= len(originalMarkers[i])
				originalMarkers[i] = nil
			}
		}
	}
	clearTopMarkers(10)

	ciMax := uint32(len(currentFile) - 1)
	oiMax := uint32(len(originalFile) - 1)

	lastCi := uint32(0)
	iterations := uint32(0)
	speedUps := 0

	maxIndex := uint32(len(currentFile)) - skew
	tt := time.Now().Add(200 * time.Millisecond)
loop:
	for ci := skew / 2; ci < maxIndex; {
		if currentFile[ci] == 0 {
			ci++
			continue
		}

		// Find most unique marker in current step
		marker := uint16((int(currentFile[ci-(skew/4)])+int(currentFile[ci-(skew/2)]))/2) | uint16((int(currentFile[ci+(skew/4)])+int(currentFile[ci+(skew/2)]))/2)<<8
		bestCi := ci
		markerLen := len(originalMarkers[marker])
		for i := uint32(0); i < step && ci+i <= maxIndex; i++ {
			if currentFile[ci+i] == 0 {
				continue
			}
			currentMarker := uint16((int(currentFile[ci+i-(skew/4)])+int(currentFile[ci+i-(skew/2)]))/2) | uint16((int(currentFile[ci+i+(skew/4)])+int(currentFile[ci+i+(skew/2)]))/2)<<8
			currentMarkerLen := len(originalMarkers[currentMarker])
			if currentMarkerLen != 0 && currentMarkerLen < markerLen {
				marker = currentMarker
				markerLen = currentMarkerLen
				bestCi = ci + i
			}
		}
		ci = bestCi

		iterations++

		if maxDuration != 0 && iterations%1000 == 0 && time.Since(ts) > maxDuration {
			break
		}

		if time.Since(tt) > 30*time.Millisecond {
			speedUps++
			if speedUps == 2 {
				step = skew * 3 / 4
			}
			if speedUps == 5 {
				step = skew
			}
			clearTopMarkers(20 + speedUps*5)
			tt = time.Now()
		}

		lastOR := uint32(0)
		nextCL := uint32(0)
		nextCR := uint32(0)
		nextOL := uint32(0)
		for _, oi := range originalMarkers[marker] {
			if lastOR >= oi ||
				currentFile[ci] != originalFile[oi] ||
				currentFile[ci+1] != originalFile[oi+1] ||
				currentFile[ci-skew/2] != originalFile[oi-skew/2] ||
				currentFile[ci+skew/2] != originalFile[oi+skew/2] {
				continue
			}
			// Validate exact range
			l, r := uint32(0), uint32(0)
			for ; oi+r < oiMax && ci+r < ciMax && originalFile[oi+r+1] == currentFile[ci+r+1]; r++ {
			}
			for ; ci-l > 0 && oi-l > 0 && originalFile[oi-l-1] == currentFile[ci-l-1]; l++ {
			}
			lastOR = oi + r
			// Determine if it's nice
			if l+r+1 >= minReuse && nextCR-nextCL < r+l {
				nextCL = ci - l
				nextCR = ci + r
				nextOL = oi - l
			}
			if l+r > reasonableReuse {
				break
			}
		}

		if nextCL != 0 || nextCR != 0 {
			addLength := int32(nextCL) - int32(lastCi)
			if addLength < 0 {
				ops.Cut(uint32(-addLength))
			} else {
				ops.Add(currentFile[lastCi:nextCL])
			}
			lastCi = nextCR + 1
			ops.Original(nextOL, nextCR-nextCL+1)
			ci = lastCi + step
			continue loop
		}

		ci += step
	}

	ops.Add(currentFile[lastCi:])

	p.buf = bytes.NewBuffer(ops.Bytes())
}

func (p *BinaryPatch) Len() int {
	return p.buf.Len()
}

func (p *BinaryPatch) Apply(original []byte) []byte {
	patch := p.buf.Bytes()
	size := binary.LittleEndian.Uint32(patch[0:4])
	result := make([]byte, size)
	resultIndex := uint32(0)
	for i := 4; i < len(patch); {
		switch patch[i] {
		case BinaryPatchOriginalOpType:
			index := binary.LittleEndian.Uint32(patch[i+1 : i+5])
			count := binary.LittleEndian.Uint32(patch[i+5 : i+9])
			copy(result[resultIndex:], original[index:index+count])
			resultIndex += count
			i += 9
		case BinaryPatchAddOpType:
			count := binary.LittleEndian.Uint32(patch[i+1 : i+5])
			copy(result[resultIndex:], patch[i+5:i+5+int(count)])
			i += 5 + int(count)
			resultIndex += count
		}
	}
	return result
}

type BinaryPatchOp struct {
	op      BinaryPatchOpType
	val1    uint32
	val2    uint32
	content []byte
}

func (b *BinaryPatchOp) Cut(bytesCount uint32) (nextOp *BinaryPatchOp, left uint32) {
	if bytesCount == 0 {
		return b, 0
	}
	switch b.op {
	case BinaryPatchOriginalOpType:
		if bytesCount >= b.val2 {
			return nil, bytesCount - b.val2
		}
		b.val2 -= bytesCount
		return b, 0
	case BinaryPatchAddOpType:
		size := uint32(len(b.content))
		if bytesCount >= size {
			return nil, bytesCount - size
		}
		b.content = b.content[0 : size-bytesCount]
		return b, 0
	}
	return nil, bytesCount
}

func (b *BinaryPatchOp) TargetSize() uint32 {
	switch b.op {
	case BinaryPatchOriginalOpType:
		return b.val2
	case BinaryPatchAddOpType:
		return uint32(len(b.content))
	}
	return 0
}

func (b *BinaryPatchOp) PatchSize() uint32 {
	switch b.op {
	case BinaryPatchOriginalOpType:
		return 9 // byte + uint32 + uint32
	case BinaryPatchAddOpType:
		return 5 + uint32(len(b.content)) // byte + uint32 + []byte(content)
	}
	return 0
}

type BinaryPatchOpList struct {
	ops   []BinaryPatchOp
	count int
}

func (b *BinaryPatchOpList) TargetSize() uint32 {
	total := uint32(0)
	for i := 0; i < b.count; i++ {
		total += b.ops[i].TargetSize()
	}
	return total
}

func (b *BinaryPatchOpList) PatchSize() uint32 {
	total := uint32(4) // uint32 for file size
	for i := 0; i < b.count; i++ {
		total += b.ops[i].PatchSize()
	}
	return total
}

func (b *BinaryPatchOpList) Cut(bytesCount uint32) uint32 {
	var next *BinaryPatchOp
	for i := b.count - 1; bytesCount > 0 && i >= 0; i-- {
		next, bytesCount = b.ops[i].Cut(bytesCount)
		if next == nil {
			b.count--
			b.ops[i] = BinaryPatchOp{}
		}
	}
	return bytesCount
}

func (b *BinaryPatchOpList) Bytes() []byte {
	targetSize := b.TargetSize()

	// Prepare buffer for the patch
	result := make([]byte, b.PatchSize())
	binary.LittleEndian.PutUint32(result, targetSize)
	resultIndex := 4

	// Include all patches
	for i := 0; i < b.count; i++ {
		switch b.ops[i].op {
		case BinaryPatchOriginalOpType:
			result[resultIndex] = BinaryPatchOriginalOpType
			binary.LittleEndian.PutUint32(result[resultIndex+1:], b.ops[i].val1)
			binary.LittleEndian.PutUint32(result[resultIndex+5:], b.ops[i].val2)
			resultIndex += 9
		case BinaryPatchAddOpType:
			result[resultIndex] = BinaryPatchAddOpType
			binary.LittleEndian.PutUint32(result[resultIndex+1:], uint32(len(b.ops[i].content)))
			copy(result[resultIndex+5:], b.ops[i].content)
			resultIndex += 5 + len(b.ops[i].content)
		}
	}
	return result
}

func (b *BinaryPatchOpList) Original(index, bytesCount uint32) {
	if bytesCount == 0 {
		return
	}

	b.append(BinaryPatchOp{op: BinaryPatchOriginalOpType, val1: index, val2: bytesCount})
}

func (b *BinaryPatchOpList) Add(bytesArr []byte) {
	if len(bytesArr) == 0 {
		return
	}

	b.append(BinaryPatchOp{op: BinaryPatchAddOpType, content: bytesArr})
}

func (b *BinaryPatchOpList) append(op BinaryPatchOp) {
	// Grow if needed
	if len(b.ops) <= b.count {
		b.ops = append(b.ops, make([]BinaryPatchOp, 100)...)
	}
	b.ops[b.count] = op
	b.count++
}
