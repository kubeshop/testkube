package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {

	t.Run("Parse Baseline Config YAML", func(t *testing.T) {
		args := Options{}
		err := args.UnmarshalYAML("../../examples/zap-baseline.yaml")

		assert.NoError(t, err)
		assert.Equal(t, "https://www.example.com/", args.Baseline.Target)
	})

	t.Run("Parse API Config YAML", func(t *testing.T) {
		args := Options{}
		err := args.UnmarshalYAML("../../examples/zap-api.yaml")

		assert.NoError(t, err)
		assert.Equal(t, "https://www.example.com/openapi.json", args.API.Target)
		assert.Equal(t, "openapi", args.API.Format)
		assert.Equal(t, "https://www.example.com", args.API.Hostname)
		assert.True(t, args.API.Safe)
		assert.True(t, args.API.Debug)
		assert.False(t, args.API.Short)
		assert.Equal(t, 5, args.API.Delay)
		assert.Equal(t, 60, args.API.Time)
		assert.Equal(t, "examples/context.config", args.API.Context)
		assert.Equal(t, "anonymous", args.API.User)
		assert.Equal(t, "-config aaa=bbb", args.API.ZapOptions)
	})

	t.Run("Parse Full Config YAML", func(t *testing.T) {
		args := Options{}
		err := args.UnmarshalYAML("../../examples/zap-full.yaml")

		assert.NoError(t, err)
		assert.Equal(t, "https://www.example.com/", args.Full.Target)
		assert.Equal(t, -1, args.Full.Minutes)
	})
}

func TestArgs(t *testing.T) {
	t.Run("Baseline Scan Args", func(t *testing.T) {
		args := Options{
			Baseline: BaselineOptions{
				Target:     "https://www.example.com/",
				Config:     "examples/zap-baseline.conf",
				Minutes:    3,
				Delay:      -1,
				Debug:      true,
				FailOnWarn: true,
				Level:      "INFO",
				Ajax:       false,
				Short:      true,
			},
		}

		cmd := args.ToBaselineScanArgs("baseline.html")

		assert.Equal(t, "-t", cmd[0])
		assert.Equal(t, "https://www.example.com/", cmd[1])
		assert.Equal(t, "-c", cmd[2])
		assert.Equal(t, "examples/zap-baseline.conf", cmd[3])
		assert.Equal(t, "-m", cmd[4])
		assert.Equal(t, "3", cmd[5])
		assert.Equal(t, "-d", cmd[6])
		assert.Equal(t, "-l", cmd[7])
		assert.Equal(t, "INFO", cmd[8])
		assert.Equal(t, "-s", cmd[9])
		assert.Equal(t, "-r", cmd[10])
		assert.Equal(t, "baseline.html", cmd[11])
		assert.Equal(t, "--auto", cmd[12])
	})

	t.Run("Full Scan Args", func(t *testing.T) {
		args := Options{
			Full: FullOptions{
				Target:     "https://www.example.com/",
				Config:     "examples/zap-baseline.conf",
				Minutes:    -1,
				Debug:      false,
				FailOnWarn: false,
				Level:      "FAIL",
				Ajax:       true,
				Short:      true,
			},
		}

		cmd := args.ToFullScanArgs("full.html")

		assert.Equal(t, "-t", cmd[0])
		assert.Equal(t, "https://www.example.com/", cmd[1])
		assert.Equal(t, "-c", cmd[2])
		assert.Equal(t, "examples/zap-baseline.conf", cmd[3])
		assert.Equal(t, "-I", cmd[4])
		assert.Equal(t, "-j", cmd[5])
		assert.Equal(t, "-l", cmd[6])
		assert.Equal(t, "FAIL", cmd[7])
		assert.Equal(t, "-s", cmd[8])
		assert.Equal(t, "-r", cmd[9])
		assert.Equal(t, "full.html", cmd[10])
	})

	t.Run("API Scan Args", func(t *testing.T) {
		args := Options{
			API: ApiOptions{
				Target:     "https://www.example.com/openapi.json",
				Format:     "openapi",
				Safe:       true,
				Config:     "https://www.example.com/zap-api.conf",
				Debug:      true,
				Short:      false,
				Level:      "PASS",
				User:       "anonymous",
				Delay:      5,
				FailOnWarn: true,
				Time:       60,
				Hostname:   "https://www.example.com",
				ZapOptions: "-config aaa=bbb",
			},
		}

		cmd := args.ToApiScanArgs("report.html")

		assert.Equal(t, "-t", cmd[0])
		assert.Equal(t, "https://www.example.com/openapi.json", cmd[1])
		assert.Equal(t, "-f", cmd[2])
		assert.Equal(t, "openapi", cmd[3])
		assert.Equal(t, "-u", cmd[4])
		assert.Equal(t, "https://www.example.com/zap-api.conf", cmd[5])
		assert.Equal(t, "-d", cmd[6])
		assert.Equal(t, "-D", cmd[7])
		assert.Equal(t, "5", cmd[8])
		assert.Equal(t, "-l", cmd[9])
		assert.Equal(t, "PASS", cmd[10])
		assert.Equal(t, "-S", cmd[11])
		assert.Equal(t, "-T", cmd[12])
		assert.Equal(t, "60", cmd[13])
		assert.Equal(t, "-U", cmd[14])
		assert.Equal(t, "anonymous", cmd[15])
		assert.Equal(t, "-O", cmd[16])
		assert.Equal(t, "https://www.example.com", cmd[17])
		assert.Equal(t, "-z", cmd[18])
		assert.Equal(t, "-config aaa=bbb", cmd[19])
	})
}
