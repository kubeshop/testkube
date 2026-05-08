package v1

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"

	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/datefilter"
	testworkflow2 "github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

const (
	exportPageSize = 100
	defaultMaxSize = 100 * 1024 * 1024 // fallback to 100 MB
)

// sequenceEntry holds the current sequence number for a workflow.
type sequenceEntry struct {
	WorkflowName string `json:"workflowName"`
	Number       int32  `json:"number"`
}

func (s *TestkubeAPI) ExportExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to export executions"

		sinceParam := c.Query("since", "")
		dFilter := datefilter.NewDateFilter(sinceParam, "")
		if sinceParam != "" && !dFilter.IsStartValid {
			return s.Error(c, http.StatusBadRequest, fmt.Errorf("invalid 'since' date: %q; expected YYYY-MM-DD or RFC 3339 format", sinceParam))
		}

		maxSize := s.exportArchiveMaxSize
		if maxSize <= 0 {
			maxSize = defaultMaxSize
		}
		archiveLimitError := fmt.Sprintf("export archive exceeds the size limit (%d bytes); use the 'since' query parameter to narrow the date range", maxSize)

		// Build archive in memory so we can enforce the size limit before sending.
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		tarWriter := tar.NewWriter(gzWriter)

		sequences := map[string]int32{}

		page := 0
		for {
			filter := testworkflow2.NewExecutionsFilter().WithPage(page).WithPageSize(exportPageSize)
			if dFilter.IsStartValid {
				filter = filter.WithStartDate(dFilter.Start)
			}

			executions, err := s.TestWorkflowResults.GetExecutions(c.Context(), filter)
			if err != nil {
				s.Log.Errorw(errPrefix+": listing executions", "error", err, "page", page)
				return s.Error(c, http.StatusInternalServerError, fmt.Errorf("listing executions: %w", err))
			}

			if len(executions) == 0 {
				break
			}

			for i := range executions {
				execution := &executions[i]

				data, err := json.Marshal(execution)
				if err != nil {
					s.Log.Errorw(errPrefix+": marshaling execution", "error", err, "id", execution.Id)
					continue
				}

				name := fmt.Sprintf("executions/%s.json", execution.Id)
				if err := writeTarEntry(tarWriter, name, data); err != nil {
					s.Log.Errorw(errPrefix+": writing execution to archive", "error", err, "id", execution.Id)
					continue
				}

				if execution.Workflow != nil {
					// Flush gzip so buf.Len() reflects the actual compressed
					// output written so far, not just what the internal gzip
					// buffer has released.
					if flushErr := gzWriter.Flush(); flushErr != nil {
						s.Log.Errorw(errPrefix+": flushing gzip writer", "error", flushErr)
						return s.Error(c, http.StatusInternalServerError, fmt.Errorf("flushing archive: %w", flushErr))
					}

					remaining := int64(maxSize - buf.Len())
					if remaining <= 0 {
						s.Log.Errorw(errPrefix+": archive size limit exceeded", "size", buf.Len())
						return s.Error(c, http.StatusRequestEntityTooLarge, errors.New(archiveLimitError))
					}

					// Read up to remaining+1 bytes so oversized logs are detected
					// by the buf.Len() > maxSize check without over-allocating.
					logData, err := s.readExecutionLog(c.Context(), execution.Id, execution.Workflow.Name, remaining+1)
					if err != nil {
						s.Log.Warnw(errPrefix+": reading logs", "error", err, "id", execution.Id)
					} else if len(logData) > 0 {
						logName := fmt.Sprintf("logs/%s.log", execution.Id)
						if err := writeTarEntry(tarWriter, logName, logData); err != nil {
							s.Log.Errorw(errPrefix+": writing log to archive", "error", err, "id", execution.Id)
						}
					}

					wfName := execution.Workflow.Name
					if execution.Number > sequences[wfName] {
						sequences[wfName] = execution.Number
					}
				}

				if buf.Len() > maxSize {
					s.Log.Errorw(errPrefix+": archive size limit exceeded", "size", buf.Len())
					return s.Error(c, http.StatusRequestEntityTooLarge, errors.New(archiveLimitError))
				}
			}

			if len(executions) < exportPageSize {
				break
			}
			page++
		}

		// Write sequences.json
		entries := make([]sequenceEntry, 0, len(sequences))
		for name, number := range sequences {
			entries = append(entries, sequenceEntry{
				WorkflowName: name,
				Number:       number,
			})
		}

		seqData, err := json.Marshal(entries)
		if err != nil {
			s.Log.Errorw(errPrefix+": marshaling sequences", "error", err)
		} else if err := writeTarEntry(tarWriter, "sequences.json", seqData); err != nil {
			s.Log.Errorw(errPrefix+": writing sequences to archive", "error", err)
		}

		if err := tarWriter.Close(); err != nil {
			s.Log.Errorw(errPrefix+": closing tar writer", "error", err)
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("finalizing archive: %w", err))
		}
		if err := gzWriter.Close(); err != nil {
			s.Log.Errorw(errPrefix+": closing gzip writer", "error", err)
			return s.Error(c, http.StatusInternalServerError, fmt.Errorf("finalizing archive: %w", err))
		}

		// Re-check size after gzip finalization — Close() flushes remaining
		// compressed data so the buffer may now exceed the limit.
		if buf.Len() > maxSize {
			s.Log.Errorw(errPrefix+": archive size limit exceeded after finalization", "size", buf.Len())
			return s.Error(c, http.StatusRequestEntityTooLarge, errors.New(archiveLimitError))
		}

		c.Set("Content-Type", "application/gzip")
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, apiclient.ExportArchiveFileName))
		return c.Send(buf.Bytes())
	}
}

func (s *TestkubeAPI) readExecutionLog(ctx context.Context, executionID, workflowName string, limit int64) ([]byte, error) {
	rc, err := s.TestWorkflowOutput.ReadLog(ctx, executionID, workflowName)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, limit))
}

// writeTarEntry writes a single file entry to a tar archive.
func writeTarEntry(tw *tar.Writer, name string, data []byte) error {
	header := &tar.Header{
		Name: name,
		Mode: 0o644,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("writing tar header for %s: %w", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("writing tar data for %s: %w", name, err)
	}
	return nil
}
