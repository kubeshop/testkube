package runner

import (
	"errors"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name           string
		scraper        func(id string, directories []string) error
		execution      testkube.Execution
		expectedError  string
		expectedStatus testkube.ExecutionStatus
	}{
		{
			name:           "successful scraper",
			scraper:        func(id string, directories []string) error { return nil },
			execution:      testkube.Execution{ArtifactRequest: &testkube.ArtifactRequest{VolumeMountPath: "."}},
			expectedError:  "",
			expectedStatus: *testkube.ExecutionStatusPassed,
		},
		{
			name:           "failing scraper",
			scraper:        func(id string, directories []string) error { return errors.New("Scraping failed") },
			execution:      testkube.Execution{ArtifactRequest: &testkube.ArtifactRequest{VolumeMountPath: "."}},
			expectedError:  "failed getting artifacts: Scraping failed",
			expectedStatus: *testkube.ExecutionStatusFailed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			s := Scraper{}
			s.ScrapeFn = test.scraper

			runner := ScraperRunner{
				ScrapperEnabled: true,
				Scraper:         s,
			}

			res, err := runner.Run(test.execution)
			assert.EqualError(t, err, test.expectedError)
			assert.Equal(t, test.expectedStatus, *res.Status)
		})
	}

}

// Scraper implements a mock for the Scraper from "github.com/kubeshop/testkube/pkg/executor/scraper"
type Scraper struct {
	ScrapeFn func(id string, directories []string) error
}

func (s Scraper) Scrape(id string, directories []string) error {
	if s.ScrapeFn == nil {
		log.Fatal("not implemented")
	}
	return s.ScrapeFn(id, directories)
}
