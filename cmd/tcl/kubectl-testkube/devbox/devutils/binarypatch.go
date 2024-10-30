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
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/dustin/go-humanize"
	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	BinaryPatchAddOp      = 1
	BinaryPatchOriginalOp = 2
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
	size := int32(30)
	step := int32(10)

	ts := time.Now()

	originalMarkers := make([][]int32, math.MaxUint16+1)
	originalIterations := int32(len(originalFile)) - size
	for i := int32(0); i < originalIterations; i++ {
		marker := uint16(originalFile[i]) | uint16(originalFile[i+size])<<8
		originalMarkers[marker] = append(originalMarkers[marker], i)
	}

	// Delete most popular characters to avoid problems with too many iterations
	lenSum := 0
	for i := uint16(0); i < math.MaxUint16; i++ {
		lenSum += len(originalMarkers[i])
	}
	lenAvg := lenSum / len(originalMarkers)
	deleted := 0
	for i := uint16(0); i < math.MaxUint16; i++ {
		if len(originalMarkers[i]) > lenAvg {
			deleted += len(originalMarkers[i])
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

	maxIndex := int32(len(currentFile)) - size
loop:
	for ci := int32(0); ci < maxIndex; ci += step {
		marker := uint16(currentFile[ci]) | uint16(currentFile[ci+size])<<8
		if len(originalMarkers[marker]) == 0 {
			ci = ci - step/2
			continue
		}
		iterations++

		if maxDuration != 0 && iterations%1000 == 0 && time.Since(ts) > maxDuration {
			break
		}

		if iterations%20000 == 0 && 100*ci/maxIndex < 50 {
			step = step * 4 / 3
			ci -= step
			continue
		}
		for _, oi := range originalMarkers[marker] {
			if (oi < step || ci < step || originalFile[oi-step] != currentFile[ci-step]) && (oi+step > oiMax || ci+step > ciMax || originalFile[oi+step] != currentFile[ci+step]) {
				continue
			}

			// Validate exact range
			l, r := int32(0), int32(0)
			for ; ci-l > lastCi && oi-l > lastOi && originalFile[oi-l-1] == currentFile[ci-l-1]; l++ {
			}
			for ; oi+r < oiMax && ci+r < ciMax && originalFile[oi+r+1] == currentFile[ci+r+1]; r++ {
			}
			// Determine if it's nice
			if l+r > 14 {
				totalSaved += int(r + l + 1)
				p.Add(currentFile[lastCi : ci-l])
				lastCi = ci + r + 1
				ci = lastCi + 1 - step
				p.Original(int(oi-l), int(r+l+1))
				continue loop
			}
		}
	}

	p.Add(currentFile[lastCi:])
}

func (p *BinaryPatch) Read333(originalFile, currentFile []byte, maxDuration time.Duration) {
	ts := time.Now()
	size := int32(32_000)

	currentMarkers := make([][]int32, math.MaxUint16+1)
	currentIterations := int32(len(currentFile)) - size
	for i := int32(0); i < currentIterations; i++ {
		marker := uint16(currentFile[i]) | uint16(currentFile[i+size])<<8
		currentMarkers[marker] = append(currentMarkers[marker], i)
	}

	// Sort all the markers
	for i := uint16(0); i < math.MaxUint16; i++ {
		slices.Sort(currentMarkers[i])
	}

	fmt.Println("indexed in", time.Since(ts))

	ciMax := int32(len(currentFile) - 1)
	oiMax := int32(len(originalFile) - 1)

	samples := make([]int32, 1000)
	for i := 0; i < 1000; i++ {
		samples[i] = int32(rand.Intn(len(originalFile) - int(size)))
	}
	slices.Sort(samples)

	lastOi := int32(0)
	lastCi := int32(0)
	totalSaved := 0

loop:
	for _, oi := range samples {
		if oi <= lastOi {
			continue
		}
		marker := uint16(originalFile[oi]) | uint16(originalFile[oi+size])<<8
		maxOL, maxOR := int32(0), int32(0)
		maxCL, maxCR := int32(0), int32(0)
		for _, ci := range currentMarkers[marker] {
			if ci <= lastCi {
				continue
			}
			// Check if it has not been validated lately
			if maxCL <= ci && maxCR >= ci {
				continue
			}

			// Validate exact range
			l, r := int32(0), int32(0)
			// Determine left side
			for ; ci-l != 0 && oi-l != 0 && originalFile[oi-l-1] == currentFile[ci-l-1]; l++ {
			}
			// Determine right side
			for ; oi+r != oiMax && ci+r != ciMax && originalFile[oi+r+1] == currentFile[ci+r+1]; r++ {
			}
			if maxOR-maxOL < r+l {
				maxOL = oi - l
				maxOR = oi + r
				maxCL = ci - l
				maxCR = ci + r
			}
			continue loop
		}

		if maxOR-maxOL > size {
			lastOi = maxOR
			lastCi = maxCR
			totalSaved += int(maxOR - maxOL + 1)
			fmt.Printf("Detected %s common (org: %d-%d, new: %d-%d)\n", humanize.Bytes(uint64(maxOR-maxOL)), maxOL, maxOR, maxCL, maxCR)
		}
	}
	fmt.Printf("Saved %s out of %s binary\n", humanize.Bytes(uint64(totalSaved)), humanize.Bytes(uint64(len(currentFile))))
}

func (p *BinaryPatch) oooRead(originalFile, currentFile []byte, maxDuration time.Duration) {
	//size := int32(min(len(originalFile)/10, len(currentFile)/10))
	size := int32(32_000)
	//pivotSize := size / 8
	ts := time.Now()

	//// Index current file by start and end
	//ts := time.Now()
	//currentZeroBytes := []int32{}
	////currentIndex := make(map[byte]map[byte]int32, 255)
	////for i := 0; i <= 255; i++ {
	////	currentIndex[byte(i)] = make(map[byte]int32)
	////}
	//for i := 0; i < len(currentFile)-size; i++ {
	//	if currentFile[i] == 0 {
	//		currentZeroBytes = append(currentZeroBytes, int32(i))
	//	}
	//	//currentIndex[currentFile[i]][currentFile[i+size]] = int32(i)
	//	//u := binary.BigEndian.Uint16([]byte{currentFile[i], currentFile[i+size]})
	//	//currentIndex[u] = append(currentIndex[u], int32(i))
	//}

	currentMarkers := make(map[uint16][]int32, math.MaxUint16+1)
	for i := uint16(0); i < math.MaxUint16; i++ {
		currentMarkers[i] = make([]int32, 0)
	}
	currentIterations := int32(len(currentFile)) - size
	for i := int32(0); i < currentIterations; i++ {
		u := uint16(currentFile[i]) | uint16(currentFile[i+size])<<8
		currentMarkers[u] = append(currentMarkers[u], i)
	}

	ciMax := int32(len(currentFile) - 1)
	oiMax := int32(len(originalFile) - 1)

	//originalMarkers := make(map[uint16][]int32, math.MaxUint16+1)
	//originalIterations := int32(len(originalFile)) - size
	//for i := int32(0); i < originalIterations; i++ {
	//	if originalFile[i] != 0 {
	//		continue
	//	}
	//	u := uint16(originalFile[i+pivotSize]) | uint16(originalFile[i+size])<<8
	//	if _, ok := originalMarkers[u]; !ok {
	//		originalMarkers[u] = []int32{i}
	//	} else {
	//		originalMarkers[u] = append(originalMarkers[u], i)
	//	}
	//}
	fmt.Println("Indexed files in", time.Since(ts))

	type set struct {
		start int32
		end   int32
	}
	commonCurrentSets := make([]set, 0)
	commonOriginalSets := make([]set, 0)

loop1:
	for i := 0; i < 10000; i++ {
		t := time.Now()
		oi := int32(rand.Intn(len(originalFile) - int(size)))
		for _, v := range commonOriginalSets {
			if v.start <= oi && v.end >= oi {
				fmt.Printf("exists. took %s\n", time.Since(t))
				continue loop1
			}
		}
		u := uint16(originalFile[oi]) | uint16(originalFile[oi+size])<<8

		maxOL, maxOR := int32(0), int32(0)
		maxCL, maxCR := int32(0), int32(0)
	loop:
		for _, ci := range currentMarkers[u] {
			// Check if it has not been validated lately
			if maxCL <= ci && maxCR >= ci {
				continue
			}
			// Check if it's not already in the common current set
			for _, v := range commonCurrentSets {
				if v.start <= ci && v.end >= ci {
					continue loop
				}
			}

			// Validate exact range
			l, r := int32(0), int32(0)
			// Determine left side
			for ; ci-l != 0 && oi-l != 0 && originalFile[oi-l-1] == currentFile[ci-l-1]; l++ {
			}
			// Determine right side
			for ; oi+r != oiMax && ci+r != ciMax && originalFile[oi+r+1] != currentFile[oi+r+1]; r++ {
			}
			if maxOR-maxOL < r+l {
				maxOL = oi - l
				maxOR = oi + r
				maxCL = ci - l
				maxCR = ci + r
			}
		}
		if maxOR-maxOL < 16000 {
			fmt.Printf("too small. took %s\n", time.Since(t))
			continue
		}
		commonCurrentSets = append(commonCurrentSets, set{start: maxCL, end: maxCR})
		commonOriginalSets = append(commonCurrentSets, set{start: maxOL, end: maxOR})
		fmt.Printf("Detected %s common (org: %d-%d, new: %d-%d)\n", humanize.Bytes(uint64(maxOR-maxOL)), maxOL, maxOR, maxCL, maxCR)
		fmt.Printf("took %s\n", time.Since(t))
	}

	//for k1, v1 := range currentMarkers {
	//	for _, i := range originalMarkers[k1] {
	//	loop:
	//		for _, j := range v1 {
	//			if originalFile[i] != currentFile[j] ||
	//				originalFile[i+size] != currentFile[j+size] ||
	//				originalFile[i+pivotSize] != currentFile[j+pivotSize] ||
	//				originalFile[i+pivotSize*2] != currentFile[j+pivotSize*2] ||
	//				originalFile[i+pivotSize*3] != currentFile[j+pivotSize*3] ||
	//				originalFile[i+pivotSize*4] != currentFile[j+pivotSize*4] ||
	//				originalFile[i+pivotSize*5] != currentFile[j+pivotSize*5] ||
	//				originalFile[i+pivotSize*6] != currentFile[j+pivotSize*6] ||
	//				originalFile[i+pivotSize*7] != currentFile[j+pivotSize*7] ||
	//				originalFile[i+size] != currentFile[j+size] {
	//				continue
	//			}
	//			for k := int32(1); k < size; k++ {
	//				if originalFile[i+k] != currentFile[j+k] {
	//					continue loop
	//				}
	//			}
	//			l, r := int32(0), size
	//			for ; i-l >= 0 && j-l >= 0; l++ {
	//				if l == 0 || originalFile[i-l-1] != currentFile[j-l-1] {
	//					break
	//				}
	//			}
	//			for ; i+r < originalIterations && j+r < currentIterations; r++ {
	//				if originalFile[i+r+1] != currentFile[j+r+1] {
	//					break
	//				}
	//			}
	//			fmt.Printf("found common %s\n", humanize.Bytes(uint64(l+size+r)))
	//		}
	//	}
	//}

	//iterations = int32(len(originalFile)) - size
	//for i := int32(0); i < iterations; i += 1 {
	//	if originalFile[i] != 0 {
	//		continue
	//	}
	//	u := uint16(255*originalFile[i] + originalFile[i+size])
	//	fmt.Println("found potential", len(currentMarkers[u]))
	//loop:
	//	for _, j := range currentMarkers[u] {
	//		if originalFile[i] != currentFile[j] ||
	//			originalFile[i+size] != currentFile[j+size] ||
	//			originalFile[i+pivotSize] != currentFile[j+pivotSize] ||
	//			originalFile[i+pivotSize*2] != currentFile[j+pivotSize*2] ||
	//			originalFile[i+pivotSize*3] != currentFile[j+pivotSize*3] ||
	//			originalFile[i+pivotSize*4] != currentFile[j+pivotSize*4] ||
	//			originalFile[i+pivotSize*5] != currentFile[j+pivotSize*5] ||
	//			originalFile[i+pivotSize*6] != currentFile[j+pivotSize*6] ||
	//			originalFile[i+pivotSize*7] != currentFile[j+pivotSize*7] ||
	//			originalFile[i+size] != currentFile[j+size] {
	//			continue
	//		}
	//		fmt.Printf("potential\n")
	//
	//		for k := int32(1); k < size; k++ {
	//			if originalFile[i+k] != currentFile[j+k] {
	//				fmt.Printf("  nope at %d\n", k)
	//				continue loop
	//			}
	//		}
	//		fmt.Printf("%x\n", currentFile[j+pivotSize-1:j+pivotSize+1])
	//		os.Exit(0)
	//	}
	//}
	//originalMarkers := make(map[byte][]int32)
	//for i := 0; i <= 255; i++ {
	//	originalMarkers[byte(i)] = make([]int32, 0)
	//}
	//for i := 0; i < len(originalFile)-size; i++ {
	//	if originalFile[i] == 0 {
	//		originalMarkers[originalFile[i+size]] = append(originalMarkers[originalFile[i+size]], int32(i))
	//	}
	//}
	//iterations = int32(len(originalFile)) - size
	//for i := int32(0); i < iterations; i++ {
	//	if originalFile[i] == 0 {
	//		for _, j := range currentMarkers[originalFile[i+size]] {
	//			fmt.Println("checking zero")
	//			if originalFile[i] != currentFile[j] ||
	//				originalFile[i+size] != currentFile[j+size] ||
	//				originalFile[i+pivotSize] != currentFile[j+pivotSize] ||
	//				originalFile[i+pivotSize*2] != currentFile[j+pivotSize*2] ||
	//				originalFile[i+pivotSize*3] != currentFile[j+pivotSize*3] ||
	//				originalFile[i+pivotSize*4] != currentFile[j+pivotSize*4] ||
	//				originalFile[i+pivotSize*5] != currentFile[j+pivotSize*5] ||
	//				originalFile[i+pivotSize*6] != currentFile[j+pivotSize*6] ||
	//				originalFile[i+pivotSize*7] != currentFile[j+pivotSize*7] ||
	//				originalFile[i+size] != currentFile[j+size] {
	//				continue
	//			}
	//			fmt.Println("found match")
	//		}
	//	}
	//}

	//for percentage := 20; percentage >= 10; percentage -= 10 {
	//	fmt.Println("Checking percentage", percentage)
	//	pivotSize := size / 8
	//	for i := 0; i < len(originalFile)-size; i++ {
	//		if i%100 == 0 {
	//			progress := 100 * i / (len(currentFile) - size)
	//			fmt.Println(progress)
	//		}
	//	loop:
	//		for j := 0; j < len(currentFile)-size; j++ {
	//			if originalFile[i] != currentFile[j] ||
	//				originalFile[i+size] != currentFile[j+size] ||
	//				originalFile[i+pivotSize] != currentFile[j+pivotSize] ||
	//				originalFile[i+pivotSize*2] != currentFile[j+pivotSize*2] ||
	//				originalFile[i+pivotSize*3] != currentFile[j+pivotSize*3] ||
	//				originalFile[i+pivotSize*4] != currentFile[j+pivotSize*4] ||
	//				originalFile[i+pivotSize*5] != currentFile[j+pivotSize*5] ||
	//				originalFile[i+pivotSize*6] != currentFile[j+pivotSize*6] ||
	//				originalFile[i+pivotSize*7] != currentFile[j+pivotSize*7] ||
	//				originalFile[i+size] != currentFile[j+size] {
	//				continue
	//			}
	//			fmt.Printf("detected potential %s block\n", humanize.Bytes(uint64(size)))
	//			for k := 1; k < size; k++ {
	//				if originalFile[i+k] != currentFile[i+k] {
	//					fmt.Printf("  nope at %d\n", k)
	//					continue loop
	//				}
	//			}
	//			fmt.Printf("detected %s block\n", humanize.Bytes(uint64(size)))
	//			os.Exit(0)
	//		}
	//	}
	//}
}

//func (p *BinaryPatch) OldRead(originalFile, currentFile []byte, maxDuration time.Duration) {
//	originalFileLen := len(originalFile)
//	currentFileLen := len(currentFile)
//	smallerFileLen := min(originalFileLen, currentFileLen)
//
//	iteration := 0
//	currentIndex := 0
//	originalIndex := 0
//
//	// Omit same sequence
//	omit := 0
//	for omit < smallerFileLen && currentFile[omit] == originalFile[omit] {
//		omit++
//	}
//	originalIndex += omit
//	currentIndex += omit
//	p.Preserve(omit)
//
//	replaced := 0
//	ts := time.Now()
//
//loop:
//	for {
//		iteration++
//		leftCurrent := currentFileLen - currentIndex
//		leftOriginal := originalFileLen - originalIndex
//		leftMin := min(leftCurrent, leftOriginal)
//		if leftMin <= BinaryPatchMinCommonSize {
//			break
//		}
//		segment := min(leftMin, BinaryBatchBlockSize*BinaryBatchBlockFactor) - BinaryPatchMinCommonSize
//		maxIterations := min(segment+replaced, leftMin)
//
//		// Extract fast when duration passed
//		if maxDuration != 0 && iteration%1000 == 0 && maxDuration < time.Since(ts) {
//			p.Delete(originalFileLen - originalIndex)
//			p.Add(currentFile[currentIndex:])
//			return
//		}
//
//		// Try recovering from an endless loop
//		if maxIterations > BinaryBatchBlockSize*BinaryBatchBlockFactorMax {
//			p.Add(currentFile[currentIndex : currentIndex+maxIterations/2])
//			replaced -= maxIterations / 2
//			currentIndex += maxIterations / 2
//			continue loop
//		}
//
//		// Find next match when adding to original file
//		biggestCommon := 0
//		biggestCommonAt := 0
//		biggestCommonDel := false
//		for i := 0; i < maxIterations; i++ {
//			common := 0
//			for i+common < leftMin && currentFile[currentIndex+i+common] == originalFile[originalIndex+common] {
//				common++
//			}
//			if common > biggestCommon {
//				biggestCommon = common
//				biggestCommonAt = i
//				biggestCommonDel = false
//			}
//			common = 0
//			for i+common < leftMin && currentFile[currentIndex+common] == originalFile[originalIndex+i+common] {
//				common++
//			}
//			if common > biggestCommon {
//				biggestCommon = common
//				biggestCommonAt = i
//				biggestCommonDel = true
//			}
//		}
//
//		if biggestCommon >= BinaryPatchMinCommonSize {
//			if biggestCommonDel {
//				p.Delete(biggestCommonAt)
//				p.Preserve(biggestCommon)
//				replaced += biggestCommonAt
//				originalIndex += biggestCommonAt + biggestCommon
//				currentIndex += biggestCommon
//			} else {
//				p.Add(currentFile[currentIndex : currentIndex+biggestCommonAt])
//				p.Preserve(biggestCommon)
//				replaced -= biggestCommonAt
//				currentIndex += biggestCommonAt + biggestCommon
//				originalIndex += biggestCommon
//			}
//			continue loop
//		}
//
//		// Treat some part as deleted to proceed
//		p.Delete(BinaryPatchMinCommonSize)
//		replaced += BinaryPatchMinCommonSize
//		originalIndex += BinaryPatchMinCommonSize
//	}
//
//	if currentIndex != currentFileLen {
//		p.Add(currentFile[currentIndex:])
//	} else if originalIndex != originalFileLen {
//		p.Preserve(originalFileLen - originalIndex)
//	}
//}

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
		//case BinaryPatchDeleteOp:
		//	count := binary.LittleEndian.Uint32(patch[i+1 : i+5])
		//	original = original[count:]
		//	i += 5
		case BinaryPatchAddOp:
			count := binary.LittleEndian.Uint32(patch[i+1 : i+5])
			result = append(result, patch[i+5:i+5+int(count)]...)
			i += 5 + int(count)
		}
	}
	return result
}
