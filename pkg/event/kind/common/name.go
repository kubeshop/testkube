package common

import (
	"log"
	"regexp"
)

// ListenerName returns name of listener which can be used by event bus to identify listener
func ListenerName(in string) string {
	reg, err := regexp.Compile("[^A-Za-z0-9.]+")
	if err != nil {
		log.Fatal(err)
	}
	return reg.ReplaceAllString(in, "")

}
