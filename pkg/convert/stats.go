package convert

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// maxReportedErrors caps how many individual failures are echoed at the end of
// a run. The counters stay exact; only the listing is truncated.
const maxReportedErrors = 10

// Stats accumulates the outcome of one migration task. Counters are only
// incremented for records that made it into a committed batch, so
// Processed+Failed+Skipped is the number of source documents actually seen --
// which on a resumed run is less than Total.
type Stats struct {
	Task string

	Total     int64 // source documents in the collection, from CountDocuments
	Processed int64
	Failed    int64
	Skipped   int64 // deliberately not migrated (e.g. non-testworkflow sequences)

	Batches int64

	Signatures           int64
	Results              int64
	Outputs              int64
	Reports              int64
	ResourceAggregations int64
	Workflows            int64

	StartTime time.Time
	EndTime   time.Time

	Errors []string
}

// Resumed is set when the task started from an existing checkpoint, so that
// Print does not report a Processed < Total run as though records were lost.
type resumeInfo struct {
	Resumed     bool
	AlreadyDone int64
}

func newStats(task string) *Stats {
	return &Stats{Task: task, StartTime: time.Now(), Errors: make([]string, 0)}
}

// addError records a failure. The counter is always bumped; the message is
// retained for the report.
func (s *Stats) addError(format string, args ...interface{}) {
	s.Failed++
	s.Errors = append(s.Errors, fmt.Sprintf(format, args...))
}

// addSkip records a document that was intentionally not migrated.
func (s *Stats) addSkip(log *zap.SugaredLogger, format string, args ...interface{}) {
	s.Skipped++
	log.Warnf(format, args...)
}

func (s *Stats) finish() {
	s.EndTime = time.Now()
}

// Duration is the wall-clock time the task took, or the time elapsed so far if
// it has not finished yet. That fallback is what lets a task defer its single
// finish call and still have Print report a sensible duration.
func (s *Stats) Duration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// Print writes a human-readable summary block. resume may be nil.
func (s *Stats) Print(log *zap.SugaredLogger, resume *resumeInfo) {
	rule := strings.Repeat("=", 70)
	d := s.Duration()

	log.Info(rule)
	log.Infof("%s MIGRATION STATISTICS", strings.ToUpper(s.Task))
	log.Info(rule)
	log.Infof("Total in source:      %d", s.Total)
	if resume != nil && resume.Resumed {
		log.Infof("Already migrated:     %d (resumed from checkpoint)", resume.AlreadyDone)
	}
	log.Infof("Processed:            %d", s.Processed)
	log.Infof("Failed:               %d", s.Failed)
	log.Infof("Skipped:              %d", s.Skipped)
	log.Infof("Batches committed:    %d", s.Batches)
	if s.Batches > 0 {
		log.Infof("Avg batch size:       %.0f", float64(s.Processed)/float64(s.Batches))
	}
	if s.Signatures+s.Results+s.Outputs+s.Reports+s.ResourceAggregations+s.Workflows > 0 {
		log.Infof("Signature rows:       %d", s.Signatures)
		log.Infof("Result rows:          %d", s.Results)
		log.Infof("Output rows:          %d", s.Outputs)
		log.Infof("Report rows:          %d", s.Reports)
		log.Infof("Resource agg rows:    %d", s.ResourceAggregations)
		log.Infof("Workflow rows:        %d", s.Workflows)
	}
	log.Infof("Duration:             %s", d.Round(time.Millisecond))
	if secs := d.Seconds(); secs > 0 {
		log.Infof("Records/second:       %.2f", float64(s.Processed)/secs)
	}
	log.Info(rule)

	if len(s.Errors) == 0 {
		return
	}

	log.Warnf("Encountered %d errors (showing up to %d):", len(s.Errors), maxReportedErrors)
	for i, e := range s.Errors {
		if i >= maxReportedErrors {
			log.Warnf("... and %d more", len(s.Errors)-maxReportedErrors)
			break
		}
		log.Warnf("  %d. %s", i+1, e)
	}
}
