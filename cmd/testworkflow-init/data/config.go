// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package data

import (
	"os"
)

type config struct {
	Negative   bool
	Debug      bool
	RetryCount int
	RetryUntil string

	Resulting []Rule
}

var Config = &config{
	Debug: os.Getenv("DEBUG") == "1",
}

func LoadConfig(config map[string]string) {
	Config.Debug = getBool(config, "debug", Config.Debug)
	Config.RetryCount = getInt(config, "retryCount", 0)
	Config.RetryUntil = getStr(config, "retryUntil", "self.passed")
	Config.Negative = getBool(config, "negative", false)
}
