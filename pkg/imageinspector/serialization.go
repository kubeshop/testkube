package imageinspector

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type Hash string

var hashKeyRe = regexp.MustCompile("[^a-zA-Z0-9-_]")

func hash(registry, image string) Hash {
	return Hash(hashKeyRe.ReplaceAllString(strings.ReplaceAll(fmt.Sprintf("%s/%s", registry, image), "/", "."), "_-"))
}

func marshalInfo(v Info) (string, error) {
	res, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func unmarshalInfo(s string) (v Info, err error) {
	err = json.Unmarshal([]byte(s), &v)
	return
}
