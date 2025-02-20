package core

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const commentPrefix = "#"

// LineParser is an interface for parsing a metric line/data points.
type LineParser interface {
	// Parse parses a single metrics line/data point.
	Parse(metric []byte) (*Metric, error)
}

// Metric represents a protocol-agnostic data point.
type Metric struct {
	Measurement string
	Tags        []KeyValue
	Fields      []KeyValue
	Timestamp   *time.Time
}

type invalidLine struct {
	Number int
	Line   string
}

func newInvalidLine(number int, line string) invalidLine {
	return invalidLine{
		Number: number,
		Line:   line,
	}
}

func ParseMetrics(ctx context.Context, reader io.Reader, filename string) ([]*Metric, []invalidLine, error) {
	scanner := bufio.NewScanner(reader)
	var line []byte
	if scanner.Scan() {
		line = scanner.Bytes()
	}
	// 1. Parse metadata from header, or if that fails, parse metadata from filename.
	metadata, err := parseMetadataFromHeader(line)
	if err != nil {
		metadata, err = parseMetadataFromFilename(filename)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse metadata from header and filename")
		}
	} else {
		line = nil
	}
	// 2. Instantiate a parser based on the metadata format.
	parser, err := NewParser(metadata.Format)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create parser")
	}

	// 3. Parse the metrics.
	return parse(metadata, parser, scanner, line)
}

// parse reads the metrics from the reader and parses them using the provided parser.
// If file header (metadata) is missing, we have already parsed a metric line, so we provide it as the last parameter.
func parse(metadata *Metadata, parser LineParser, scanner *bufio.Scanner, unaccountedMetric []byte) ([]*Metric, []invalidLine, error) {
	var metrics []*Metric
	var add bool
	if metadata.Lines > 0 {
		add = true
		metrics = make([]*Metric, metadata.Lines)
	} else {
		add = false
		metrics = []*Metric{}
	}

	var invalidLines []invalidLine
	i := -1
	if len(unaccountedMetric) > 0 {
		i++
		line := strings.TrimSpace(string(unaccountedMetric))
		metric, err := parser.Parse([]byte(line))
		if err != nil {
			invalidLines = append(invalidLines, newInvalidLine(i+1, string(unaccountedMetric)))
		} else {
			addOrAppend(&metrics, metric, i, add)
		}
	}

	for scanner.Scan() {
		i++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines or commented lines
		if line == "" || strings.HasPrefix(line, commentPrefix) {
			continue
		}

		metric, err := parser.Parse([]byte(line))
		if err != nil {
			invalidLines = append(invalidLines, newInvalidLine(i+1, line))
			continue
		}
		addOrAppend(&metrics, metric, i, add)
	}
	if err := scanner.Err(); err != nil {
		return nil, invalidLines, errors.Wrap(err, "error while reading metrics file")
	}

	return metrics, invalidLines, nil
}

func addOrAppend(metrics *[]*Metric, m *Metric, i int, add bool) {
	if add {
		(*metrics)[i] = m
	} else {
		*metrics = append(*metrics, m)
	}

}

// NewParser is a factory method which instantiates a parser implementation based on the provided format.
func NewParser(format MetricsFormat) (LineParser, error) {
	switch format {
	case FormatInflux:
		return NewInfluxDBLineProtocolParser(), nil
	default:
		return nil, errors.Errorf("unsupported format: %s", format)
	}
}

type InfluxDBLineProtocolParser struct{}

func NewInfluxDBLineProtocolParser() *InfluxDBLineProtocolParser {
	return &InfluxDBLineProtocolParser{}
}

func (p *InfluxDBLineProtocolParser) Parse(sample []byte) (*Metric, error) {
	parsed, err := p.parse(string(sample))
	if err != nil {
		return nil, err
	}

	return &Metric{
		Measurement: parsed.Measurement,
		Tags:        parsed.Tags,
		Fields:      parsed.Fields,
		Timestamp:   parsed.Timestamp,
	}, nil
}

// parse parses a single line of InfluxDB line protocol
// into a Metric structure. This function is a simplified parser
// that does not handle all escaping or quoting rules of the full
// line protocol specification.
func (p *InfluxDBLineProtocolParser) parse(line string) (*Metric, error) {
	// Split up to 3 parts: [metric+tags, fields, timestamp]
	parts := strings.SplitN(strings.TrimSpace(line), " ", 3)
	if len(parts) < 2 {
		return nil, errors.New("invalid line protocol: must have at least measurement+tags and fields")
	}

	// The first part is "measurement[,tag1=value1,tag2=value2,...]"
	measurementAndTags := parts[0]
	// The second part is "field1=value1,field2=value2,..."
	fieldsPart := parts[1]

	// The third part is the optional timestamp.
	var timestampStr string
	if len(parts) == 3 {
		timestampStr = parts[2]
	}

	metric, tags, err := p.parseMeasurementAndTags(measurementAndTags)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	fields, err := p.parseFields(fieldsPart)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ts, err := p.parseTimestamp(timestampStr)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Metric{
		Measurement: metric,
		Tags:        tags,
		Fields:      fields,
		Timestamp:   ts,
	}, nil
}

// parseMeasurementAndTags parses the measurement name and optional tags.
// Example: "cpu,host=server01,region=uswest"
func (p *InfluxDBLineProtocolParser) parseMeasurementAndTags(input string) (string, []KeyValue, error) {
	idx := strings.IndexRune(input, ',')
	if idx == -1 {
		// No tags, everything is the measurement
		return input, nil, nil
	}

	// Measurement is everything before first comma
	measurement := input[:idx]

	// Tag string is after first comma
	tagStr := input[idx+1:]
	tagPairs := strings.Split(tagStr, ",")

	var tags []KeyValue
	for _, pair := range tagPairs {
		kvParts := strings.SplitN(pair, "=", 2)
		if len(kvParts) != 2 {
			return "", nil, errors.Errorf("invalid tag: %s", pair)
		}
		key := kvParts[0]
		val := kvParts[1]
		tags = append(tags, NewKeyValue(key, val))
	}
	return measurement, tags, nil
}

// parseFields parses the fields string.
// Example: "field1=value1,field2=value2,..."
func (p *InfluxDBLineProtocolParser) parseFields(input string) ([]KeyValue, error) {
	if input == "" {
		return nil, errors.New("fields part is empty")
	}
	fieldPairs := strings.Split(input, ",")
	var fields []KeyValue
	for _, pair := range fieldPairs {
		kvParts := strings.SplitN(pair, "=", 2)
		if len(kvParts) != 2 {
			return nil, fmt.Errorf("invalid field: %s", pair)
		}
		key := kvParts[0]
		val := kvParts[1]
		fields = append(fields, NewKeyValue(key, val))
	}
	return fields, nil
}

// parseTimestamp parses the optional timestamp string with nano precision.
// Example: "1739525804000000000"
func (p *InfluxDBLineProtocolParser) parseTimestamp(timestamp string) (*time.Time, error) {
	var ts *time.Time
	if timestamp != "" {
		nano, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			return nil, errors.Errorf("invalid timestamp: %v", err)
		}
		_ts := time.Unix(0, nano).UTC()
		ts = &_ts
	}
	return ts, nil
}

var _ LineParser = &InfluxDBLineProtocolParser{}
