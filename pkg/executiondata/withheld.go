package executiondata

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// withheldMarkerPrefix opens the marker published in place of an output whose real
// value may not leave the workflow that produced it.
const withheldMarkerPrefix = "<testkube:withheld output "

var withheldMarkerRe = regexp.MustCompile(withheldMarkerPrefix + `[^>]*>`)

// WithheldMarker is the value a step publishes in place of an output it produced but
// would not send outside its own workflow.
//
// Outputs are published by printing an instruction to the log stream, which is
// obfuscated on its way out, so a value holding a sensitive word would reach the
// execution record partially masked and its reader would silently use something
// corrupted. Publishing this marker instead keeps the real value inside the workflow
// that produced it, and keeps the omission loud: a consumer resolves the marker
// rather than an empty value, and the steps handing data to another workflow refuse
// to run with it.
func WithheldMarker(workflow, name string) string {
	if workflow == "" {
		return fmt.Sprintf("%s%s>", withheldMarkerPrefix, name)
	}
	return fmt.Sprintf("%s%s of workflow %s>", withheldMarkerPrefix, name, workflow)
}

// IsWithheldMarker reports whether a string carries a withheld marker. It is the cheap
// check for callers holding a single value already - WithheldMarkersIn walks a whole
// structure, and is what to call once this says there is something to name.
func IsWithheldMarker(value string) bool {
	return strings.Contains(value, withheldMarkerPrefix)
}

// WithheldMarkersIn lists the withheld markers the strings of a value carry,
// deduplicated and in the order they are found.
//
// The value is walked as JSON, so it may be a whole resolved specification - a marker
// may sit anywhere a workflow author could have written an expression. A value JSON
// cannot represent carries no markers.
func WithheldMarkersIn(value interface{}) []string {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var walked interface{}
	if err := json.Unmarshal(raw, &walked); err != nil {
		return nil
	}
	markers := make([]string, 0)
	collectWithheldMarkers(walked, &markers, map[string]struct{}{})
	return markers
}

func collectWithheldMarkers(value interface{}, markers *[]string, seen map[string]struct{}) {
	switch v := value.(type) {
	case string:
		for _, marker := range withheldMarkerRe.FindAllString(v, -1) {
			if _, ok := seen[marker]; ok {
				continue
			}
			seen[marker] = struct{}{}
			*markers = append(*markers, marker)
		}
	case []interface{}:
		for i := range v {
			collectWithheldMarkers(v[i], markers, seen)
		}
	case map[string]interface{}:
		// Maps are walked in a stable order, so the same specification always reports
		// its markers the same way.
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			collectWithheldMarkers(v[key], markers, seen)
		}
	}
}

// WithheldError explains that something cannot run because it reads an output that
// was never published. It names every marker it was given, so the author can see
// which output of which workflow to stop relying on.
func WithheldError(subject string, markers []string) error {
	return fmt.Errorf("%s reads an output that was not published outside the workflow that produced it (%s): its value holds a sensitive word, and publishing it would have corrupted it on the way through the obfuscated log stream - exchange it as an artifact read with %s(), or through a secret both workflows can reach",
		subject, strings.Join(markers, ", "), ReadArtifactFn)
}
