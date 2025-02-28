package core

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// MetricsFormat defines the allowed formats ("influx", "csv", "json").
type MetricsFormat string

const (
	prefixMeta                           = "META "
	FormatInflux           MetricsFormat = "influx"
	FormatCSV              MetricsFormat = "csv"
	FormatJSON             MetricsFormat = "json"
	FormatUnknown          MetricsFormat = "unknown"
	maxStringSize                        = 50
	metadataFieldSeparator               = "."
)

var (
	ErrNoMetadata          = errors.New("no metadata found")
	ErrInvalidMetadataLine = errors.New("invalid metadata line")
)

// Metadata holds the parsed result from the meta line.
type Metadata struct {
	Workflow           string             `meta:"workflow"`
	Step               Step               `meta:"step"`
	Execution          string             `meta:"execution"`
	Lines              int                `meta:"lines"`
	Format             MetricsFormat      `meta:"format"`
	ContainerResources ContainerResources `meta:"resources"`
}

type Step struct {
	Ref  string `meta:"ref"`
	Name string `meta:"name,quote"`
}

type ContainerResources struct {
	Requests ResourceList `meta:"requests"`
	Limits   ResourceList `meta:"limits"`
}

type ResourceList struct {
	CPU    string `meta:"cpu"`
	Memory string `meta:"memory"`
}

// String uses reflection on `meta` tags to produce a string.
func (m *Metadata) String() string {
	sb := strings.Builder{}
	sb.WriteString(prefixMeta)

	encodeStruct(reflect.ValueOf(m).Elem(), &sb, "") // no prefix at top-level
	return strings.TrimSpace(sb.String())
}

type metadataTag struct {
	name  string
	quote bool
}

// parseMetaTag parses a meta tag like "step,quote" => { name: "step", quote: true }.
func parseMetaTag(tag string) (metadataTag, error) {
	parts := strings.Split(tag, ",")
	mt := metadataTag{
		name: parts[0], // always at least one part
	}
	// For each additional option, set a bool accordingly.
	for _, opt := range parts[1:] {
		switch opt {
		case "quote":
			mt.quote = true
		default:
			return mt, errors.Errorf("unrecognized meta tag option %q", opt)
		}
	}
	return mt, nil
}

// encodeStruct handles struct fields by looking for `meta` tags and dispatching to encodeValue as needed.
func encodeStruct(v reflect.Value, sb *strings.Builder, prefix string) {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)

		tagVal := fieldType.Tag.Get("meta")
		if tagVal == "" {
			// No meta tag, skip or recurse if you want deeper nest logic
			continue
		}

		// Parse the meta tag to get the field name and any options
		mt, _ := parseMetaTag(tagVal)

		// If the field is a pointer, we'll dereference it, otherwise the address of the field will be returned.
		if fieldVal.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				continue
			}
			fieldVal = fieldVal.Elem()
		}
		// If the field is itself another struct, we might want to recurse.
		if fieldVal.Kind() == reflect.Struct {
			// prefix becomes e.g. "resources."
			encodeStruct(fieldVal, sb, prefix+tagVal+metadataFieldSeparator)
			continue
		}

		// It's a "normal" field (string, int, etc.).
		// We'll skip zero values (empty string, 0, etc.) so we don't clutter the output.
		if isZero(fieldVal) {
			continue
		}

		// Build final key for this field. If we have a prefix, include it.
		// e.g. if prefix="resources.requests." and tagVal="cpu", finalKey = "resources.requests.cpu"
		finalKey := prefix + mt.name

		// Build the value to print
		var strVal string
		// If the "quote" option is on, we’ll wrap the value in quotes.
		// (Typically we only do that if it's a string, but you can decide otherwise.)
		if mt.quote && fieldVal.Kind() == reflect.String {
			// strconv.Quote will escape any internal quotes, newlines, etc.
			// e.g. "foo" => "\"foo\""
			// If you'd prefer raw quotes, do: strVal = `"` + fieldVal.String() + `"`
			strVal = strconv.Quote(fieldVal.String())
		} else {
			// fallback
			strVal = fmt.Sprintf("%v", fieldVal.Interface())
		}

		// Write out "key=value "
		sb.WriteString(finalKey)
		sb.WriteByte('=')
		sb.WriteString(strVal)
		sb.WriteByte(' ')
	}
}

// isZero checks if a reflect.Value is the "zero value" for its type
// (0 for ints, "" for strings, nil for pointers, etc.)
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0.0
	case reflect.Bool:
		return v.Bool() == false
	case reflect.Slice, reflect.Array:
		return v.Len() == 0
	case reflect.Map, reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Struct:
		return false
	default:
		return false
	}
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
		Step:      Step{Ref: tokens[1]},
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
	length := bytes.IndexByte(header, 0x00)
	metadataBuf := header[metadataStartIndex:min(len(header), length)]

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

	tokens := split(remaining)
	if len(tokens) == 0 {
		return &Metadata{}, nil
	}

	var (
		lines    int
		format   MetricsFormat
		stepName string
	)

	m := Metadata{}

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
			m.Lines = lines
		case "format":
			format = MetricsFormat(value)
			switch format {
			case FormatInflux, FormatCSV, FormatJSON:
				// valid
			default:
				return nil, errors.Wrapf(ErrInvalidMetadataLine, "unsupported metrics format %q", format)
			}
			m.Format = format
		case "workflow":
			m.Workflow = value
		case "step.ref":
			m.Step.Ref = value
		case "step.name":
			stepName, err = strconv.Unquote(value)
			if err != nil {
				return nil, errors.Wrapf(ErrInvalidMetadataLine, "failed to unquote 'step.name' in %q", value)
			}
			m.Step.Name = stepName
		case "execution":
			m.Execution = value
		case "resources.requests.cpu":
			m.ContainerResources.Requests.CPU = value
		case "resources.requests.memory":
			m.ContainerResources.Requests.Memory = value
		case "resources.limits.cpu":
			m.ContainerResources.Limits.CPU = value
		case "resources.limits.memory":
			m.ContainerResources.Limits.Memory = value
		default:
			return nil, errors.Errorf("unrecognized metadata key %q in token %q", key, token)
		}
	}

	return &m, nil
}
