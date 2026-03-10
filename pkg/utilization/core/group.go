package core

import (
	"sort"
	"time"
)

type GroupedMetrics struct {
	KeyMap map[string][]string
	Data   []*DataPoint
}

type fieldsByMeasurement map[string]valuesByField
type valuesByField map[string]*valuesWrapper
type valuesWrapper struct {
	Tags   []keyValueTuple
	Values []timestampValueTuple
}

// DataPoint represents an optimized JSON structure for transferring metrics.
type DataPoint struct {
	Measurement string                `json:"measurement"`
	Tags        []keyValueTuple       `json:"tags"`
	Field       string                `json:"fields"`
	Values      []timestampValueTuple `json:"values"`
}

// timestampValueTuple is a tuple of timestamp and value.
type timestampValueTuple [2]any

// toTimestampValueTuple converts a timestamp and a value to a timestampValueTuple.
func toTimestampValueTuple(timestamp time.Time, value any) timestampValueTuple {
	return timestampValueTuple{timestamp.UTC().UnixMilli(), value}
}

// keyValueTuple is a tuple of key and value strings.
type keyValueTuple [2]string

func keyValuesToTuples(keyValues []KeyValue) []keyValueTuple {
	tags := make([]keyValueTuple, 0, len(keyValues))
	for _, kv := range keyValues {
		tags = append(tags, toKeyValueTuple(kv.Key, kv.Value))
	}
	return tags
}

// toKeyValueTuple converts a key and a value to a keyValueTuple.
func toKeyValueTuple(key, value string) keyValueTuple {
	return keyValueTuple{key, value}
}

// GroupMetrics converts a slice of metrics to a slice of JSON data points optimized for transfer.
func GroupMetrics(metrics []*Metric) GroupedMetrics {
	grouped := groupDataPoints(metrics)
	sortDataPoints(grouped)
	dataPoints := buildDataPoints(grouped)
	keymap := buildKeyMap(grouped)
	return GroupedMetrics{
		KeyMap: keymap,
		Data:   dataPoints,
	}
}

// buildKeyMap builds a map of measurement to fields for the given metrics.
// Example: {"cpu": ["usage", "idle"], "memory": ["used", "free"]}
func buildKeyMap(fieldsMap fieldsByMeasurement) map[string][]string {
	keymap := make(map[string][]string)
	for measurement, fields := range fieldsMap {
		if _, ok := keymap[measurement]; !ok {
			keymap[measurement] = make([]string, 0, len(fields))
		}
		for field := range fields {
			keymap[measurement] = append(keymap[measurement], field)
		}
	}
	return keymap
}

func buildDataPoints(fieldsMap fieldsByMeasurement) []*DataPoint {
	var dataPoints []*DataPoint
	for measurement, fields := range fieldsMap {
		for field, metricWrapper := range fields {
			point := &DataPoint{
				Measurement: measurement,
				Tags:        metricWrapper.Tags,
				Field:       field,
				Values:      metricWrapper.Values,
			}
			dataPoints = append(dataPoints, point)
		}
	}
	return dataPoints
}

func groupDataPoints(metrics []*Metric) fieldsByMeasurement {
	fieldsMap := make(fieldsByMeasurement)
	for _, metric := range metrics {
		if metric == nil {
			continue
		}
		// Skip metrics without a timestamp
		if metric.Timestamp == nil {
			continue
		}
		for _, field := range metric.Fields {
			if _, ok := fieldsMap[metric.Measurement]; !ok {
				fieldsMap[metric.Measurement] = make(valuesByField, len(metric.Fields))
			}
			valuesMap := fieldsMap[metric.Measurement]
			if _, ok := valuesMap[field.Key]; !ok {
				w := &valuesWrapper{
					Tags:   keyValuesToTuples(metric.Tags),
					Values: make([]timestampValueTuple, 0, len(metric.Fields)),
				}
				fieldsMap[metric.Measurement][field.Key] = w
			}
			w := fieldsMap[metric.Measurement][field.Key]
			w.Values = append(w.Values, toTimestampValueTuple(*metric.Timestamp, field.Value))
		}
	}
	return fieldsMap
}

// sortDataPoints sorts the values of each field by timestamp in ascending order.
func sortDataPoints(fieldsMap fieldsByMeasurement) {
	for _, fields := range fieldsMap {
		for _, wrapper := range fields {
			sort.Slice(wrapper.Values, func(i, j int) bool {
				return wrapper.Values[i][0].(int64) < wrapper.Values[j][0].(int64)
			})

		}
	}
}
