package parser

import (
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/ui"
)

type CSVResults struct {
	HasError         bool
	LastErrorMessage string
	Results          []CSVResult
}

type CSVResult struct {
	Success      bool
	Error        string
	Label        string
	ResponseCode string
	Duration     time.Duration
}

func parseCSV(reader io.Reader) (results CSVResults, err error) {
	res, err := CSVToMap(reader)
	if err != nil {
		return
	}

	for _, r := range res {
		result := MapElementToResult(r)
		results.Results = append(results.Results, result)

		if !result.Success {
			results.HasError = true
			results.LastErrorMessage = result.Error
		}
	}

	return
}

// CSVToMap takes a reader and returns an array of dictionaries, using the header row as the keys
func CSVToMap(reader io.Reader) ([]map[string]string, error) {
	r := csv.NewReader(reader)
	var rows []map[string]string
	var header []string
	for {
		record, err := r.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if header == nil {
			header = record
		} else {
			dict := map[string]string{}
			for i := range header {
				dict[header[i]] = record[i]
			}
			rows = append(rows, dict)
		}
	}
	return rows, nil
}

func MapElementToResult(in map[string]string) CSVResult {
	elapsed, err := strconv.Atoi(in["elapsed"])
	if err != nil {
		output.PrintLogf("%s Error parsing elapsed time to int from JTL report: %s", ui.IconWarning, err.Error())
	}

	return CSVResult{
		Success:      in["success"] == "true",
		Error:        in["failureMessage"],
		Label:        in["label"],
		Duration:     time.Millisecond * time.Duration(elapsed),
		ResponseCode: in["responseCode"],
	}
}
