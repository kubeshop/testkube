package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type Formatter interface {
	Format(metric string, tags []KeyValue, fields []KeyValue) string
}

type KeyValue struct {
	Key   string
	Value string
}

func NewKeyValue(key, value string) KeyValue {
	return KeyValue{
		Key:   key,
		Value: value,
	}
}

// NewFormatter is a factory method which instantiates a formatter implementation based on the provided format.
func NewFormatter(format MetricsFormat) (Formatter, error) {
	switch format {
	case FormatInflux:
		return NewInfluxDBLineProtocolFormatter(), nil
	default:
		return nil, errors.Errorf("unsupported format: %s", format)
	}
}

type InfluxDBLineProtocolFormatter struct {
	now func() time.Time
}

func NewInfluxDBLineProtocolFormatter() *InfluxDBLineProtocolFormatter {
	return &InfluxDBLineProtocolFormatter{
		now: time.Now,
	}
}

func (f *InfluxDBLineProtocolFormatter) Format(metric string, tags []KeyValue, fields []KeyValue) string {
	sb := strings.Builder{}
	// Metrics
	sb.WriteString(metric)
	// Tags
	if len(tags) > 0 {
		sb.WriteString(",")
		sb.WriteString(f.formatKeyValue(tags))
	}
	// Whitespace
	sb.WriteString(" ")
	// Fields
	if len(fields) > 0 {
		sb.WriteString(f.formatKeyValue(fields))
		sb.WriteString(" ")
	}
	// Timestamp
	sb.WriteString(fmt.Sprintf("%d", f.now().UnixNano()))

	return sb.String()
}

func (f *InfluxDBLineProtocolFormatter) formatKeyValue(kv []KeyValue) string {
	sb := strings.Builder{}
	for i, v := range kv {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%s=%s", v.Key, v.Value))
	}
	return sb.String()
}

var _ Formatter = &InfluxDBLineProtocolFormatter{}
