package core

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// MetricsFormat defines the allowed formats ("influx", "csv", "json").
type MetricsFormat string

const (
	prefixMeta                  = "#META "
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

// WriteMetadataToFile writes the metadata at the end of the file.
func WriteMetadataToFile(f *os.File, metadata *Metadata) error {
	_, err := f.WriteString(metadata.String() + "\n")
	if err != nil {
		return errors.Wrap(err, "failed to write metadata to the file")
	}
	return nil
}

// parseMetadataFromFile reads the first line of the file (header) and tries to parse it as Metadata.
// This function always rewinds the file pointer to the beginning after reading the header.
func parseMetadataFromFile(f *os.File) (*Metadata, error) {
	header, _, err := readLastLine(f)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer f.Seek(0, 0)

	metadata, err := parseMetadata(string(header))
	if err != nil {
		if !errors.Is(err, ErrNoMetadata) {
			return nil, errors.WithStack(err)
		}
	}
	// If metadata is not nil, we have successfully parsed it from the file header.
	if metadata != nil {
		// If the format is not set, we need to determine it based on the file extension.
		if metadata.Format == "" {
			metadata.Format, err = getFormatFromFileExtension(f.Name())
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}
		// We can return the metadata now.
		return metadata, nil
	}
	// If metadata is nil, we need to determine the file format based on the file extension.
	ext, err := getFormatFromFileExtension(f.Name())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &Metadata{Format: ext}, nil
}

func getFormatFromFileExtension(path string) (MetricsFormat, error) {
	ext := filepath.Ext(path)
	switch ext {
	case ".csv":
		return FormatCSV, nil
	case ".json":
		return FormatJSON, nil
	case ".influx":
		return FormatInflux, nil
	default:
		return FormatUnknown, errors.Errorf("unsupported file format: %s", ext)
	}
}

// parseMetadata parses a line and checks does it contain metadata.
// Example format:
//
//	"#META lines=<int> format=<influx|csv|json>"
//
// It returns a Metadata struct or an error if the line is malformed.
func parseMetadata(line string) (*Metadata, error) {
	// Must start with "#META "
	if !strings.HasPrefix(line, prefixMeta) {
		return nil, errors.Wrapf(ErrNoMetadata, "meta line must start with %q, got: %q", prefixMeta, line)
	}

	// Remove "#META " prefix
	remaining := strings.TrimPrefix(line, prefixMeta)
	// remaining is e.g. "lines=10 format=influx"

	// Split by spaces (we expect exactly 2 tokens: "lines=NN" and "format=XXX")
	tokens := strings.Fields(remaining)
	if len(tokens) == 0 {
		return &Metadata{}, nil
	}

	var (
		linesStr     string
		formatStr    string
		workflowStr  string
		stepStr      string
		executionStr string
	)

	// We'll parse each token, which should be in "key=value" form.
	for _, token := range tokens {
		kv := strings.SplitN(token, "=", 2)
		if len(kv) != 2 {
			return nil, errors.Wrapf(ErrInvalidMetadataLine, "invalid key=value pair: %q", token)
		}
		key, value := kv[0], kv[1]
		switch key {
		case "lines":
			linesStr = value
		case "format":
			formatStr = value
		case "workflow":
			workflowStr = value
		case "step":
			stepStr = value
		case "execution":
			executionStr = value
		default:
			return nil, errors.Errorf("unrecognized metadata key %q in token %q", key, token)
		}
	}

	// Now parse lines=<int>
	linesInt, err := strconv.Atoi(linesStr)
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidMetadataLine, "failed to parse 'lines' as int in %q", linesStr)
	}

	// Check the format value
	switch MetricsFormat(formatStr) {
	case FormatInflux, FormatCSV, FormatJSON:
		// valid
	default:
		return nil, errors.Wrapf(ErrInvalidMetadataLine, "unsupported format %q; must be one of [influx, csv, json]", formatStr)
	}

	meta := &Metadata{
		Lines:     linesInt,
		Format:    MetricsFormat(formatStr),
		Workflow:  workflowStr,
		Step:      stepStr,
		Execution: executionStr,
	}
	return meta, nil
}
