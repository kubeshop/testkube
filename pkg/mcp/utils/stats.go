package utils

import (
	"math"
	"sort"
)

// Stats holds computed statistics for a series of values.
type Stats struct {
	Count  int     `json:"count"`
	Mean   float64 `json:"mean"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	StdDev float64 `json:"stdDev"`
}

// ComputeStats calculates statistics for a slice of float64 values.
// Returns zero values if the slice is empty.
func ComputeStats(values []float64) Stats {
	n := len(values)
	if n == 0 {
		return Stats{}
	}

	// Calculate mean, min, max
	var sum, min, max float64
	min = values[0]
	max = values[0]

	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	mean := sum / float64(n)

	// Calculate standard deviation
	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(n)
	stdDev := math.Sqrt(variance)

	return Stats{
		Count:  n,
		Mean:   mean,
		Min:    min,
		Max:    max,
		StdDev: stdDev,
	}
}

// ZScore calculates the z-score (standard score) for a value.
// Returns 0 if stdDev is 0 to avoid division by zero.
func ZScore(value, mean, stdDev float64) float64 {
	if stdDev == 0 {
		return 0
	}
	return (value - mean) / stdDev
}

// IsOutlier returns true if the z-score exceeds the threshold (typically 2.0).
func IsOutlier(value, mean, stdDev, threshold float64) bool {
	if stdDev == 0 {
		return false
	}
	z := ZScore(value, mean, stdDev)
	return math.Abs(z) > threshold
}

// Trend represents the direction of change in a metric over time.
type Trend string

const (
	TrendIncreasing Trend = "increasing"
	TrendDecreasing Trend = "decreasing"
	TrendStable     Trend = "stable"
)

// DetectTrend analyzes values over time to determine the overall trend.
// Uses linear regression slope to determine trend direction.
// Values are assumed to be in chronological order (oldest first).
func DetectTrend(values []float64) Trend {
	n := len(values)
	if n < 3 {
		return TrendStable
	}

	// Simple linear regression to find slope
	// x values are 0, 1, 2, ..., n-1
	var sumX, sumY, sumXY, sumX2 float64
	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	nf := float64(n)
	denominator := nf*sumX2 - sumX*sumX
	if denominator == 0 {
		return TrendStable
	}

	slope := (nf*sumXY - sumX*sumY) / denominator

	// Calculate the mean of values to determine relative significance of slope
	mean := sumY / nf
	if mean == 0 {
		// If mean is zero, use absolute threshold
		if slope > 0.01 {
			return TrendIncreasing
		} else if slope < -0.01 {
			return TrendDecreasing
		}
		return TrendStable
	}

	// Relative slope (as percentage of mean per execution)
	relativeSlope := slope / mean

	// Threshold: if slope changes > 1% of mean per execution, it's a trend
	if relativeSlope > 0.01 {
		return TrendIncreasing
	} else if relativeSlope < -0.01 {
		return TrendDecreasing
	}
	return TrendStable
}

// Percentile calculates the p-th percentile of a slice of values.
// p should be between 0 and 100.
func Percentile(values []float64, p float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}

	// Sort a copy to avoid modifying the original
	sorted := make([]float64, n)
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate the index
	index := (p / 100) * float64(n-1)
	lower := int(index)
	upper := lower + 1

	if upper >= n {
		return sorted[n-1]
	}

	// Linear interpolation between lower and upper
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
