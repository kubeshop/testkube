package v1

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"

	testworkflow2 "github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

// sequenceEntry holds the current sequence number for a workflow.
type sequenceEntry struct {
	WorkflowName string `json:"workflowName"`
	Number       int32  `json:"number"`
}

func (s *TestkubeAPI) ExportExecutionsHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		errPrefix := "failed to export executions"

		c.Set("Content-Type", "application/gzip")
		c.Set("Content-Disposition", `attachment; filename="testkube-export.tar.gz"`)

		// Capture the request context for cancellation propagation.
		// This is safe because SendStream blocks until the pipe reader is fully consumed,
		// so the context remains valid while the goroutine runs.
		reqCtx := c.Context()

		pr, pw := io.Pipe()

		go func() {
			var writeErr error
			defer func() { pw.CloseWithError(writeErr) }()

			gzWriter := gzip.NewWriter(pw)
			tarWriter := tar.NewWriter(gzWriter)

			// Track max sequence number per workflow
			sequences := map[string]int32{}

			page := 0
			pageSize := 100
			for {
				filter := testworkflow2.NewExecutionsFilter().WithPage(page).WithPageSize(pageSize)
				executions, err := s.TestWorkflowResults.GetExecutions(reqCtx, filter)
				if err != nil {
					s.Log.Errorw(errPrefix+": listing executions", "error", err, "page", page)
					writeErr = fmt.Errorf("listing executions: %w", err)
					return
				}

				if len(executions) == 0 {
					break
				}

				for i := range executions {
					execution := &executions[i]

					// Write execution metadata as JSON to executions/<id>.json
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

					// Read logs from MinIO and write to logs/<id>.log
					if execution.Workflow != nil {
						logData, err := s.readExecutionLog(reqCtx, execution.Id, execution.Workflow.Name)
						if err != nil {
							s.Log.Warnw(errPrefix+": reading logs", "error", err, "id", execution.Id)
						} else if len(logData) > 0 {
							logName := fmt.Sprintf("logs/%s.log", execution.Id)
							if err := writeTarEntry(tarWriter, logName, logData); err != nil {
								s.Log.Errorw(errPrefix+": writing log to archive", "error", err, "id", execution.Id)
							}
						}

						// Track max sequence per workflow
						wfName := execution.Workflow.Name
						if execution.Number > sequences[wfName] {
							sequences[wfName] = execution.Number
						}
					}
				}

				if len(executions) < pageSize {
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

			// Close writers in order to flush all data
			if err := tarWriter.Close(); err != nil {
				s.Log.Errorw(errPrefix+": closing tar writer", "error", err)
				writeErr = err
			}
			if err := gzWriter.Close(); err != nil {
				s.Log.Errorw(errPrefix+": closing gzip writer", "error", err)
				if writeErr == nil {
					writeErr = err
				}
			}
		}()

		return c.SendStream(pr)
	}
}

func (s *TestkubeAPI) readExecutionLog(ctx context.Context, executionID, workflowName string) ([]byte, error) {
	rc, err := s.TestWorkflowOutput.ReadLog(ctx, executionID, workflowName)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
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
