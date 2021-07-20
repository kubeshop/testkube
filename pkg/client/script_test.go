package main

import (
	"testing"
)

func TestGetCRDEmptiness(t *testing.T) {

	var test = SriptsFromCRD{
		name: map[string]string{
			"First":  "firstValue",
			"Second": "secondValue",
			"Third":  "thirdValue",
		},
	}

	ans, _ := test.Get("First")

	if ans == "" {
		t.Error("Get() returned empty string")
	}
}

func TestGetCRDCorrectness(t *testing.T) {

	var test = SriptsFromCRD{
		name: map[string]string{
			"First":  "firstValue",
			"Second": "secondValue",
			"Third":  "thirdValue",
		},
	}

	ans, _ := test.Get("First")

	if ans != "firstValue" {
		t.Error("Get() returned incorrect string. Expecting, ", "firstValue. Returned", ans)
	}
}
