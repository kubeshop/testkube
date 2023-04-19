package runner

import (
	"context"
	"errors"
	"log"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/kubeshop/testkube/pkg/executor/scraper"

	"github.com/kubeshop/testkube/pkg/envs"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestRun(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	e := testkube.Execution{ArtifactRequest: &testkube.ArtifactRequest{VolumeMountPath: "."}}
	tests := []struct {
		name           string
		scraper        func(id string, directories []string) error
		execution      testkube.Execution
		expectedError  string
		expectedStatus *testkube.ExecutionStatus
		scraperBuilder func() scraper.Scraper
	}{
		{
			name:           "successful scraper",
			scraper:        func(id string, directories []string) error { return nil },
			execution:      e,
			expectedError:  "",
			expectedStatus: nil,
			scraperBuilder: func() scraper.Scraper {
				s := scraper.NewMockScraper(mockCtrl)
				s.EXPECT().Scrape(gomock.Any(), []string{"."}, gomock.Eq(e)).Return(nil)
				s.EXPECT().Close().Return(nil)
				return s
			},
		},
		{
			name:           "failing scraper",
			scraper:        func(id string, directories []string) error { return errors.New("Scraping failed") },
			execution:      testkube.Execution{ArtifactRequest: &testkube.ArtifactRequest{VolumeMountPath: "."}},
			expectedError:  "error scraping artifacts from container executor: Scraping failed",
			expectedStatus: testkube.ExecutionStatusFailed,
			scraperBuilder: func() scraper.Scraper {
				s := scraper.NewMockScraper(mockCtrl)
				s.EXPECT().Scrape(gomock.Any(), []string{"."}, gomock.Eq(e)).Return(errors.New("Scraping failed"))
				s.EXPECT().Close().Return(nil)
				return s
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			runner := ScraperRunner{
				Params:  envs.Params{ScrapperEnabled: true},
				Scraper: test.scraperBuilder(),
			}

			res, err := runner.Run(context.Background(), test.execution)
			if err != nil {
				assert.EqualError(t, err, test.expectedError)
				assert.Equal(t, *test.expectedStatus, *res.Status)
			} else {
				assert.Empty(t, test.expectedError)
				assert.Empty(t, res.Status)
			}
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
