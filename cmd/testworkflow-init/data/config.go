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
