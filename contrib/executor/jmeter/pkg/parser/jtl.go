package parser

import (
	"encoding/csv"
	"errors"
	"io"
	"log"
	"strconv"
	"time"
)

type Results struct {
	HasError         bool
	LastErrorMessage string
	Results          []Result
}

type Result struct {
	Success      bool
	Error        string
	Label        string
	ResponseCode string
	Duration     time.Duration
}

func Parse(reader io.Reader) (results Results) {
	res := CSVToMap(reader)
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

func MapElementToResult(in map[string]string) Result {
	elapsed, _ := strconv.Atoi(in["elapsed"])

	return Result{
		Success:      in["success"] == "true",
		Error:        in["failureMessage"],
		Label:        in["label"],
		Duration:     time.Millisecond * time.Duration(elapsed),
		ResponseCode: in["responseCode"],
	}
}

// CSVToMap takes a reader and returns an array of dictionaries, using the header row as the keys
func CSVToMap(reader io.Reader) []map[string]string {
	r := csv.NewReader(reader)
	rows := []map[string]string{}
	var header []string
	for {
		record, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatal(err)
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
	return rows
}
