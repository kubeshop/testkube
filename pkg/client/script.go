package main

import (
	"errors"
	"fmt"
)

// SriptsFromCRD. Struct which will hold Scripts returned via k8s API calls.
type SriptsFromCRD struct {
	name map[string]string
}

// Get. Function for returning newman CRD script. Accepts ScriptName. Returng value(string).
func (s *SriptsFromCRD) Get(scriptName string) (string, error) {

	// checking if there is no empty name on ScriptsFromCRD
	if s.name[scriptName] == "" {
		return "", errors.New("empty name")
	}
	fmt.Println("Found ", s.name[scriptName])
	return s.name[scriptName], nil
}

// GetScriptsAPI. Returns SriptsFromCRD struct from k8s API.
// func GetScriptsAPI(kubeClient kube.Client) ScriptsFromCRD {
// 	return nil
// }

func main() {

	var test = SriptsFromCRD{
		name: map[string]string{
			"First":  "firstValue",
			"Second": "secondValue",
			"Third":  "thirdValue",
		},
	}

	name, _ := test.Get("First")
	fmt.Println("Reading with the Get() method: ", name)
}
