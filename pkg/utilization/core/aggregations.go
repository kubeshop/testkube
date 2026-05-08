package core

import (
	"math"
	"strconv"

	"github.com/pkg/errors"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

// GroupedAggregations is a map of field name to Aggregation.
type GroupedAggregations map[string]map[string]*AggregationWrapper

type AggregationWrapper struct {
	// count is the number of data points.
	count int
	// m2 is an auxiliary variable used to calculate standard deviation.
	m2 float64
	testworkflowsv1.TestWorkflowExecutionResourceAggregations
}

func CalculateAggregations(metrics []*DataPoint) GroupedAggregations {
	aggregations := make(GroupedAggregations)

	for _, m := range metrics {
		if aggregations[m.Measurement] == nil {
			aggregations[m.Measurement] = make(map[string]*AggregationWrapper)
		}
		if aggregations[m.Measurement][m.Field] == nil {
			aggregations[m.Measurement][m.Field] = newDefaultAggregation()
		}
		a := aggregations[m.Measurement][m.Field]
		calculate(m.Values, a)
	}

	for _, byField := range aggregations {
		for _, a := range byField {
			calculateStandardDeviation(a)
		}
	}

	return aggregations
}

func calculate(values []timestampValueTuple, a *AggregationWrapper) {
	for _, v := range values {
		val, err := parseValue(v[1])
		if err != nil {
			return
		}
		a.count++
		if val > 0 {
			a.Min = math.Min(a.Min, val)
		}
		a.Max = math.Max(a.Max, val)
		a.Total += val
		// Use Welford formula to calculate variance.
		// More info: https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance
		delta := val - a.Avg
		a.Avg += delta / float64(a.count)
		delta2 := val - a.Avg
		a.m2 += delta * delta2
	}
	// if there is no data points, set min to 0
	if a.Min == math.MaxFloat64 {
		a.Min = 0
	}
}

func parseValue(val any) (float64, error) {
	switch v := val.(type) {
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, errors.Errorf("unsupported value type: %T", v)
	}
}

func calculateStandardDeviation(a *AggregationWrapper) {
	a.StdDev = 0
	if a.count > 1 {
		variance := a.m2 / float64(a.count-1) // sample variance
		a.StdDev = math.Sqrt(variance)
	}
}

func newDefaultAggregation() *AggregationWrapper {
	return &AggregationWrapper{
		TestWorkflowExecutionResourceAggregations: testworkflowsv1.TestWorkflowExecutionResourceAggregations{
			Min: math.MaxFloat64,
		},
	}
}
