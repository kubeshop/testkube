package testresults

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// defaultInternalPath mirrors cmd/testworkflow-init/constants. It is repeated
	// rather than imported because the toolkit binary must resolve the same
	// directory as the init binary, and neither may import the other's command
	// package. The environment variable is what the integration test framework
	// overrides, so both sides have to honour it.
	defaultInternalPath = "/.tktw"

	// handoffDirName is the directory under the internal volume that holds one
	// file per step that reached a verdict.
	handoffDirName = "testcases"
)

// ReportVerdict is what the container that decided a step's verdict leaves
// behind for the container that uploads that step's report.
//
// The two are different containers. The verdict is decided in the init process
// milliseconds after the test tool exits, because only the pod can still change
// the step's status; the report is uploaded later by the toolkit, from the
// artifacts stage. Nothing in the report itself records that a failure was
// muted - muting is a property of the workflow's policy, not of the XML - so
// without this the uploaded report says a test failed and cannot say that the
// failure was expected.
//
// The two cannot be joined by step reference: the artifacts stage is given a
// reference of its own by the processor, distinct from the run stage's. They
// are joined by report file path instead, which is exact - the file the verdict
// was read from is the same file being uploaded.
type ReportVerdict struct {
	// Reports are the absolute paths of the report files this verdict was read
	// from. A step may name several; the verdict covers their merge.
	Reports []string `json:"reports,omitempty"`
	// Muted are the canonical ids of the failing test cases the mute patterns
	// covered.
	Muted []string `json:"muted,omitempty"`
	// Unexpected is how many failing test cases they did not cover.
	Unexpected int32 `json:"unexpected,omitempty"`
	// Tolerated is whether the step met its pass requirement.
	Tolerated bool `json:"tolerated,omitempty"`
}

// HandoffDir is the directory the verdicts are exchanged through.
func HandoffDir() string {
	internal := os.Getenv("TESTKUBE_TW_INTERNAL_PATH")
	if internal == "" {
		internal = defaultInternalPath
	}
	return filepath.Join(internal, handoffDirName)
}

// WriteVerdict records one step's verdict for the container that will upload
// its report.
//
// Failing to write is worth reporting to the caller but not worth failing the
// step over: the verdict itself has already been applied to the step status,
// and what is lost is the muted counts on the uploaded report.
func WriteVerdict(ref string, verdict ReportVerdict) error {
	dir := HandoffDir()
	if err := os.MkdirAll(dir, 0777); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	payload, err := json.Marshal(verdict)
	if err != nil {
		return fmt.Errorf("encoding the verdict: %w", err)
	}

	path := filepath.Join(dir, verdictFileName(ref))
	if err := os.WriteFile(path, payload, 0666); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// ReadVerdicts collects every verdict recorded in this pod, keyed by the report
// path each one covers.
//
// A missing directory is the ordinary case rather than an error: almost no step
// declares a testCases policy, and one that does not reaches no verdict.
// Individual files that cannot be read are skipped for the same reason a report
// that cannot be parsed is still uploaded - a broken handoff should cost the
// muted counts, not the artifact.
func ReadVerdicts() (map[string]ReportVerdict, error) {
	dir := HandoffDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	verdicts := map[string]ReportVerdict{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		payload, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var verdict ReportVerdict
		if err := json.Unmarshal(payload, &verdict); err != nil {
			continue
		}
		for _, report := range verdict.Reports {
			verdicts[filepath.Clean(report)] = verdict
		}
	}
	return verdicts, nil
}

// verdictFileName keeps a step reference usable as a file name. References are
// generated identifiers and already safe, so this only guards against a shape
// changing under us later.
func verdictFileName(ref string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, ref)
	if safe == "" {
		safe = "step"
	}
	return safe + ".json"
}
