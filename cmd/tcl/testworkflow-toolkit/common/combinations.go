// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package common

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"golang.org/x/exp/maps"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
)

func CountCombinations(matrix map[string][]interface{}) int64 {
	combinations := int64(1)
	for k := range matrix {
		combinations *= int64(len(matrix[k]))
	}
	return combinations
}

func GetMatrixValues(matrix map[string][]interface{}, index int64) map[string]interface{} {
	// Compute modulo for each matrix parameter
	keys := maps.Keys(matrix)
	modulo := map[string]int64{}
	floor := map[string]int64{}
	for i, k := range keys {
		modulo[k] = int64(len(matrix[k]))
		floor[k] = 1
		for j := i + 1; j < len(keys); j++ {
			floor[k] *= int64(len(matrix[keys[j]]))
		}
	}

	// Compute values for selected index
	result := make(map[string]interface{})
	for _, k := range keys {
		kIdx := (index / floor[k]) % modulo[k]
		result[k] = matrix[k][kIdx]
	}
	return result
}

func ReadCount(s intstr.IntOrString, machines ...expressions.Machine) (int64, error) {
	countExpr, err := expressions.CompileAndResolve(s.String(), machines...)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid: %s", s.String(), err)
	}
	if countExpr.Static() == nil {
		return 0, fmt.Errorf("%s: could not resolve: %s", s.String(), err)
	}
	countVal, err := countExpr.Static().IntValue()
	if err != nil {
		return 0, fmt.Errorf("%s: could not convert to int: %s", s.String(), err)
	}
	if countVal < 0 {
		return 0, fmt.Errorf("%s: should not be lower than zero", s.String())
	}
	return countVal, nil
}

func readParams(base map[string]testworkflowsv1.DynamicList, machines ...expressions.Machine) (map[string][]interface{}, error) {
	result := make(map[string][]interface{})
	for key, items := range base {
		exprStr := items.Expression
		if !items.Dynamic {
			statics := items.DeepCopy().Static
			err := expressions.FinalizeForce(&statics, machines...)
			if err != nil {
				return nil, fmt.Errorf("%s: error while resolving matrix: %s", key, err)
			}
			b, err := json.Marshal(statics)
			if err != nil {
				return nil, fmt.Errorf("%s: could not parse list of values: %s", key, err)
			}
			exprStr = string(b)
		}
		expr, err := expressions.CompileAndResolve(exprStr, machines...)
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %s", key, exprStr, err)
		}
		if expr.Static() == nil {
			return nil, fmt.Errorf("%s: %s: could not resolve", key, exprStr)
		}
		list, err := expr.Static().SliceValue()
		if err != nil {
			return nil, fmt.Errorf("%s: %s: could not parse as list: %s", key, exprStr, err)
		}
		result[key] = list
	}
	for key := range result {
		if len(result[key]) == 0 {
			delete(result, key)
		}
	}
	return result, nil
}

func GetShardValues(values map[string][]interface{}, index int64, count int64) map[string][]interface{} {
	result := make(map[string][]interface{})
	for k := range values {
		if index > int64(len(values[k])) {
			result[k] = []interface{}{}
			continue
		}
		shards := int64(len(values[k]))
		size := int64(math.Floor(float64(shards) / float64(count)))
		if index >= shards {
			result[k] = make([]interface{}, 0)
			continue
		}
		sizeMatchPoint := shards - size*count
		if sizeMatchPoint > index {
			start := index * (size + 1)
			end := start + (size + 1)
			result[k] = values[k][start:end]
		} else {
			start := sizeMatchPoint + index*size
			end := start + size
			result[k] = values[k][start:end]
		}
	}
	return result
}

type ParamsSpec struct {
	ShardCount  int64
	MatrixCount int64
	Count       int64
	Matrix      map[string][]interface{}
	Shards      map[string][]interface{}
}

func (p *ParamsSpec) String(parallelism int64) string {
	infos := make([]string, 0)
	if p.MatrixCount > 1 {
		infos = append(infos, fmt.Sprintf("%d combinations", p.MatrixCount))
	}
	if p.ShardCount > 1 {
		infos = append(infos, fmt.Sprintf("sharded %d times", p.ShardCount))
	}
	if p.Count > 1 {
		if parallelism < p.Count {
			if parallelism == 1 {
				infos = append(infos, fmt.Sprintf("parallel: %d", parallelism))
			} else {
				infos = append(infos, "run sequentially")
			}
		} else if parallelism >= p.Count {
			infos = append(infos, "all in parallel")
		}
	}
	if p.Count == 1 {
		return "1 instance requested"
	}
	return fmt.Sprintf("%d instances requested: %s", p.Count, strings.Join(infos, ", "))
}

func (p *ParamsSpec) ShardIndexAt(index int64) int64 {
	return index % p.ShardCount
}

func (p *ParamsSpec) MatrixIndexAt(index int64) int64 {
	return (index - p.ShardIndexAt(index)) / p.ShardCount
}

func (p *ParamsSpec) ShardsAt(index int64) map[string][]interface{} {
	return GetShardValues(p.Shards, p.ShardIndexAt(index), p.ShardCount)
}

func (p *ParamsSpec) MatrixAt(index int64) map[string]interface{} {
	return GetMatrixValues(p.Matrix, p.MatrixIndexAt(index))
}

func (p *ParamsSpec) MachineAt(index int64) expressions.Machine {
	// Get basic indices
	shardIndex := p.ShardIndexAt(index)
	matrixIndex := p.MatrixIndexAt(index)

	// Compute values for this instance
	matrixValues := p.MatrixAt(index)
	shardValues := p.ShardsAt(index)

	return expressions.NewMachine().
		Register("index", index).
		Register("count", p.Count).
		Register("matrixIndex", matrixIndex).
		Register("matrixCount", p.MatrixCount).
		Register("matrix", matrixValues).
		Register("shardIndex", shardIndex).
		Register("shardCount", p.ShardCount).
		Register("shard", shardValues)
}

func (p *ParamsSpec) Humanize() string {
	// Print information
	infos := make([]string, 0)
	if p.MatrixCount > 1 {
		infos = append(infos, fmt.Sprintf("%d combinations", p.MatrixCount))
	}
	if p.ShardCount > 1 {
		infos = append(infos, fmt.Sprintf("sharded %d times", p.ShardCount))
	}
	if p.Count == 0 {
		return "no executions requested"
	}
	if p.Count == 1 {
		return "1 execution requested"
	}
	return fmt.Sprintf("%d executions requested: %s", p.Count, strings.Join(infos, ", "))
}

func GetParamsSpec(origMatrix map[string]testworkflowsv1.DynamicList, origShards map[string]testworkflowsv1.DynamicList, origCount *intstr.IntOrString, origMaxCount *intstr.IntOrString, machines ...expressions.Machine) (*ParamsSpec, error) {
	// Resolve the shards and matrix
	shards, err := readParams(origShards, machines...)
	if err != nil {
		return nil, fmt.Errorf("shards: %w", err)
	}
	matrix, err := readParams(origMatrix, machines...)
	if err != nil {
		return nil, fmt.Errorf("matrix: %w", err)
	}
	minShards := int64(math.MaxInt64)
	for key := range shards {
		if int64(len(shards[key])) < minShards {
			minShards = int64(len(shards[key]))
		}
	}

	// Calculate number of matrix combinations
	combinations := CountCombinations(matrix)

	// Resolve the count
	var count, maxCount *int64
	if origCount != nil {
		countVal, err := ReadCount(*origCount, machines...)
		if err != nil {
			return nil, fmt.Errorf("count: %w", err)
		}
		count = &countVal
	}
	if origMaxCount != nil {
		countVal, err := ReadCount(*origMaxCount, machines...)
		if err != nil {
			return nil, fmt.Errorf("maxCount: %w", err)
		}
		maxCount = &countVal
	}
	if count == nil && maxCount == nil {
		count = common.Ptr(int64(1))
	}
	if count != nil && maxCount != nil && *maxCount < *count {
		count = maxCount
		maxCount = nil
	}
	if maxCount != nil && *maxCount > minShards {
		count = &minShards
	} else if maxCount != nil {
		count = maxCount
	}

	return &ParamsSpec{
		ShardCount:  *count,
		MatrixCount: combinations,
		Count:       *count * combinations,
		Matrix:      matrix,
		Shards:      shards,
	}, nil
}
