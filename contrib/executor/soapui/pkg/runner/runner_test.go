package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestRun_Integration(t *testing.T) {
	test.IntegrationTest(t)
	t.Skipf("Skipping integration test %s until it is installed in CI", t.Name())

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tempDir, err := os.MkdirTemp("", "*")
	assert.NoErrorf(t, err, "failed to create temp dir: %v", err)
	defer os.RemoveAll(tempDir)

	testXML := "./example/REST-Project-1-soapui-project.xml"
	writeTestContent(t, tempDir, testXML)

	e := testkube.Execution{
		Id:            "get_petstore",
		TestName:      "Get Petstore",
		TestNamespace: "petstore",
		TestType:      "soapui/xml",
		Name:          "Testing GET",
		Args:          []string{"-c 'TestCase 1'"},
		Content:       testkube.NewStringTestContent(""),
	}

	tests := []struct {
		name            string
		testFileCreator func() (*os.File, error)
		execution       testkube.Execution
		expectedError   string
		expectedStatus  testkube.ExecutionStatus
		scraperBuilder  func() scraper.Scraper
	}{
		{
			name:            "Successful test, successful scraper",
			testFileCreator: createSuccessfulScript,
			execution:       e,
			expectedError:   "",
			expectedStatus:  *testkube.ExecutionStatusPassed,
			scraperBuilder: func() scraper.Scraper {
				s := scraper.NewMockScraper(mockCtrl)
				s.EXPECT().Scrape(gomock.Any(), []string{"/logs"}, []string{}, gomock.Eq(e)).Return(nil)
				return s
			},
		},
		{
			name:            "Failing test, successful scraper",
			testFileCreator: createFailingScript,
			execution:       e,
			expectedError:   "",
			expectedStatus:  *testkube.ExecutionStatusFailed,
			scraperBuilder: func() scraper.Scraper {
				s := scraper.NewMockScraper(mockCtrl)
				s.EXPECT().Scrape(gomock.Any(), []string{"/logs"}, []string{}, gomock.Eq(e)).Return(nil)
				return s
			},
		},
		{
			name:            "Successful test, failing scraper",
			testFileCreator: createSuccessfulScript,
			execution:       e,
			expectedError:   "error scraping artifacts from SoapUI executor: Scraping failed",
			expectedStatus:  *testkube.ExecutionStatusPassed,
			scraperBuilder: func() scraper.Scraper {
				s := scraper.NewMockScraper(mockCtrl)
				s.EXPECT().Scrape(gomock.Any(), []string{"/logs"}, []string{}, gomock.Eq(e)).Return(errors.New("Scraping failed"))
				return s
			},
		},
		{
			name:            "Failing test, failing scraper",
			testFileCreator: createFailingScript,
			execution:       e,
			expectedError:   "error scraping artifacts from SoapUI executor: Scraping failed",
			expectedStatus:  *testkube.ExecutionStatusFailed,
			scraperBuilder: func() scraper.Scraper {
				s := scraper.NewMockScraper(mockCtrl)
				s.EXPECT().Scrape(gomock.Any(), []string{"/logs"}, []string{}, gomock.Eq(e)).Return(errors.New("Scraping failed"))
				return s
			},
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			file, err := test.testFileCreator()
			assert.NoError(t, err)
			defer file.Close()
			params := envs.Params{DataDir: tempDir, ScrapperEnabled: true}
			runner := SoapUIRunner{
				SoapUILogsPath: "/logs",
				Params:         params,
				Scraper:        test.scraperBuilder(),
			}

			test.execution.Command = []string{
				"/bin/sh",
				file.Name(),
			}
			test.execution.Args = []string{
				"<runPath>",
			}
			res, err := runner.Run(context.Background(), test.execution)
			if test.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, test.expectedError)
			}

			assert.Equal(t, test.expectedStatus, *res.Status)
		})
	}
}

func createSuccessfulScript() (*os.File, error) {
	file, err := os.CreateTemp("", "successful_script")
	if err != nil {
		return nil, err
	}

	_, err = file.WriteString("#!/bin/sh\nexit 0\n")
	if err != nil {
		return nil, err
	}

	return file, nil
}

func createFailingScript() (*os.File, error) {
	file, err := os.CreateTemp("", "failing_script")
	if err != nil {
		return nil, err
	}

	_, err = file.WriteString("#!/bin/sh\nexit 1\n")
	if err != nil {
		return nil, err
	}

	return file, nil
}

func writeTestContent(t *testing.T, dir string, testScript string) {
	soapuiScript, err := os.ReadFile(testScript)
	if err != nil {
		assert.FailNow(t, "Unable to read soapui test script")
	}

	err = os.WriteFile(filepath.Join(dir, "test-content"), soapuiScript, 0644)
	if err != nil {
		assert.FailNow(t, "Unable to write soapui runner test content file")
	}
}
