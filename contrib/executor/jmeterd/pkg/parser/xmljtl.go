package parser

import (
	"encoding/xml"
)

// XMLResults is a root element of junit xml report
type XMLResults struct {
	XMLName     xml.Name  `xml:"testResults"`
	HTTPSamples []Example `xml:"httpSample,omitempty"`
	Samples     []Example `xml:"sample,omitempty"`
}

// Example is example details
type Example struct {
	Time            int              `xml:"t,attr"`
	Success         bool             `xml:"s,attr"`
	Label           string           `xml:"lb,attr"`
	ResponseCode    string           `xml:"rc,attr"`
	AssertionResult *AssertionResult `xml:"assertionResult"`
}

// AssertionResult contains assertion
type AssertionResult struct {
	XMLName        xml.Name `xml:"assertionResult"`
	Name           string   `xml:"name"`
	Failure        bool     `xml:"failure"`
	Error          bool     `xml:"error"`
	FailureMessage string   `xml:"failureMessage"`
}

func parseXML(data []byte) (results XMLResults, err error) {
	err = xml.Unmarshal(data, &results)

	return results, err
}
