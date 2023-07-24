package parser

import (
	"encoding/xml"
)

// Testsuites is a root element of junit report
type Testsuites struct {
	XMLName    xml.Name    `xml:"testsuites"`
	Testsuites []Testsuite `xml:"testsuite,omitempty"`
	Name       string      `xml:"name,attr,omitempty"`
	Tests      int         `xml:"tests,attr,omitempty"`
	Failures   int         `xml:"failures,attr,omitempty"`
	Errors     int         `xml:"errors,attr,omitempty"`
	Skipped    int         `xml:"skipped,attr,omitempty"`
	Assertions int         `xml:"assertions,attr,omitempty"`
	Time       float32     `xml:"time,attr,omitempty"`
	Timestamp  string      `xml:"timestamp,attr,omitempty"`
}

// Testsuite contains testsuite definition
type Testsuite struct {
	XMLName    xml.Name   `xml:"testsuite"`
	Testcases  []Testcase `xml:"testcase"`
	Name       string     `xml:"name,attr"`
	Tests      int        `xml:"tests,attr"`
	Failures   int        `xml:"failures,attr"`
	Errors     int        `xml:"errors,attr"`
	Skipped    int        `xml:"skipped,attr,omitempty"`
	Assertions int        `xml:"assertions,attr,omitempty"`
	Time       float32    `xml:"time,attr"`
	Timestamp  string     `xml:"timestamp,attr,omitempty"`
	File       string     `xml:"file,attr,omitempty"`
}

// TestResult represents the result of a testcase
type TestResult struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr,omitempty"`
}

// Testcase define a testcase
type Testcase struct {
	XMLName    xml.Name    `xml:"testcase"`
	Name       string      `xml:"name,attr"`
	ClassName  string      `xml:"classname,attr"`
	Assertions int         `xml:"assertions,attr,omitempty"`
	Time       float32     `xml:"time,attr,omitempty"`
	File       string      `xml:"file,attr,omitempty"`
	Line       int         `xml:"line,attr,omitempty"`
	Skipped    *TestResult `xml:"skipped,omitempty"`
	Failure    *TestResult `xml:"failure,omitempty"`
	Error      *TestResult `xml:"error,omitempty"`
}
