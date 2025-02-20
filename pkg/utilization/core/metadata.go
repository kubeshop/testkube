package core

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// MetricsFormat defines the allowed formats ("influx", "csv", "json").
type MetricsFormat string

const (
	prefixMeta                  = "META "
	FormatInflux  MetricsFormat = "influx"
	FormatCSV     MetricsFormat = "csv"
	FormatJSON    MetricsFormat = "json"
	FormatUnknown MetricsFormat = "unknown"
)

var (
	ErrNoMetadata          = errors.New("no metadata found")
	ErrInvalidMetadataLine = errors.New("invalid metadata line")
)

// Metadata holds the parsed result from the meta line.
type Metadata struct {
	Workflow  string
	Step      string
	Execution string
	Lines     int
	Format    MetricsFormat
}

func (m *Metadata) String() string {
	sb := strings.Builder{}
	sb.WriteString(prefixMeta)
	if m.Workflow != "" {
		sb.WriteString("workflow=")
		sb.WriteString(m.Workflow)
		sb.WriteString(" ")
	}
	if m.Step != "" {
		sb.WriteString("step=")
		sb.WriteString(m.Step)
		sb.WriteString(" ")
	}
	if m.Execution != "" {
		sb.WriteString("execution=")
		sb.WriteString(m.Execution)
		sb.WriteString(" ")
	}
	if m.Lines > 0 {
		sb.WriteString("lines=")
		sb.WriteString(strconv.Itoa(m.Lines))
		sb.WriteString(" ")
	}
	if m.Format != "" {
		sb.WriteString("format=")
		sb.WriteString(string(m.Format))
	}
	return sb.String()
}

func parseMetadataFromFilename(filename string) (*Metadata, error) {
	base := filepath.Base(filename)
	format := getFormatFromFileExtension(filename)
	if format == FormatUnknown {
		return nil, errors.Errorf("unsupported metrics file extension %q", filename)
	}
	base = strings.TrimSuffix(base, filepath.Ext(base))
	tokens := strings.Split(base, "_")
	if len(tokens) != 3 {
		return nil, errors.Errorf("invalid filename format: expected <workflow>_<step>_<execution>.<format>, got: %q", base)
	}
	return &Metadata{
		Workflow:  tokens[0],
		Step:      tokens[1],
		Execution: tokens[2],
		Format:    format,
	}, nil
}

func getFormatFromFileExtension(filename string) MetricsFormat {
	ext := filepath.Ext(filename)
	switch ext {
	case ".csv":
		return FormatCSV
	case ".json":
		return FormatJSON
	case ".influx":
		return FormatInflux
	default:
		return FormatUnknown
	}
}

// parseMetadataFromHeader checks is the provided header valid and extracts metadata from it.
func parseMetadataFromHeader(header []byte) (*Metadata, error) {
	controlByte := header[metadataControlByteIndex]
	if controlByte != metadataControlByte {
		return nil, errors.Wrapf(ErrNoMetadata, "invalid header control byte %q", controlByte)
	}
	length := header[metadataLengthByteIndex]
	metadataBuf := header[metadataStartIndex : metadataStartIndex+length]

	return parseMetadata(string(metadataBuf))
}

// parseMetadata parses a line and checks does it contain metadata.
// Example format:
//
//	"#META lines=<int> format=<influx|csv|json>"
//
// It returns a Metadata struct or an error if the line is malformed.
func parseMetadata(line string) (*Metadata, error) {
	if !strings.HasPrefix(line, prefixMeta) {
		return nil, errors.Wrapf(ErrNoMetadata, "meta line must start with %q, got: %q", prefixMeta, line)
	}

	// Remove "#META " prefix
	remaining := strings.TrimPrefix(line, prefixMeta)
	// remaining is e.g. "lines=10 format=influx"

	tokens := strings.Fields(remaining)
	if len(tokens) == 0 {
		return &Metadata{}, nil
	}

	var (
		lines     int
		format    MetricsFormat
		workflow  string
		step      string
		execution string
	)

	var err error
	// We'll parse each token, which should be in "key=value" form.
	for _, token := range tokens {
		kv := strings.SplitN(token, "=", 2)
		if len(kv) != 2 {
			return nil, errors.Wrapf(ErrInvalidMetadataLine, "invalid key=value pair: %q", token)
		}
		key, value := kv[0], kv[1]
		switch key {
		case "lines":
			lines, err = strconv.Atoi(value)
			if err != nil {
				return nil, errors.Wrapf(ErrInvalidMetadataLine, "failed to parse 'lines' as int in %q", value)
			}
		case "format":
			format = MetricsFormat(value)
			switch format {
			case FormatInflux, FormatCSV, FormatJSON:
				// valid
			default:
				return nil, errors.Wrapf(ErrInvalidMetadataLine, "unsupported metrics format %q", format)
			}
		case "workflow":
			workflow = value
		case "step":
			step = value
		case "execution":
			execution = value
		default:
			return nil, errors.Errorf("unrecognized metadata key %q in token %q", key, token)
		}
	}

	meta := &Metadata{
		Lines:     lines,
		Format:    format,
		Workflow:  workflow,
		Step:      step,
		Execution: execution,
	}
	return meta, nil
}
