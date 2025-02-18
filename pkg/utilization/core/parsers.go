package core

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const commentPrefix = "#"

// Parser is an interface for parsing metrics/data points.
type Parser interface {
	// Parse parses a single metrics line/data point.
	Parse(sample []byte) (*Sample, error)
}

// Sample represents a single data point.
type Sample struct {
	Metric    string
	Tags      []KeyValue
	Fields    []KeyValue
	Timestamp *time.Time
}

func ParseMetricsFile(filepath string) ([]*Sample, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open metrics file")
	}
	defer f.Close()

	metadata, err := parseMetadataFromFile(f)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse metadata")
	}

	parser, err := newParser(metadata.Format)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create parser")
	}

	var samples []*Sample
	if metadata.Lines > 0 {
		samples = make([]*Sample, metadata.Lines)
	}
	scanner := bufio.NewScanner(f)
	i := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines or commented lines
		if line == "" || strings.HasPrefix(line, commentPrefix) {
			continue
		}

		sample, err := parser.Parse([]byte(line))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse line %d: %q", i+1, line)
		}
		samples[i] = sample
		i++
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "error while reading metrics file")
	}

	return samples, nil
}

func GroupByMetric(samples []*Sample) map[string][]*Sample {
	grouped := make(map[string][]*Sample)
	for _, s := range samples {
		grouped[s.Metric] = append(grouped[s.Metric], s)
	}
	return grouped
}

func newParser(format FormatType) (Parser, error) {
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

func (p *InfluxDBLineProtocolParser) Parse(sample []byte) (*Sample, error) {
	parsed, err := p.parse(string(sample))
	if err != nil {
		return nil, err
	}

	return &Sample{
		Metric:    parsed.Metric,
		Tags:      parsed.Tags,
		Fields:    parsed.Fields,
		Timestamp: parsed.Timestamp,
	}, nil
}

// parse parses a single line of InfluxDB line protocol
// into a Sample structure. This function is a simplified parser
// that does not handle all escaping or quoting rules of the full
// line protocol specification.
func (p *InfluxDBLineProtocolParser) parse(line string) (*Sample, error) {
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

	return &Sample{
		Metric:    metric,
		Tags:      tags,
		Fields:    fields,
		Timestamp: ts,
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

var _ Parser = &InfluxDBLineProtocolParser{}
