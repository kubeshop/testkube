package runner

import (
	"fmt"
	"os"
	"strings"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"
)

type ApiOptions struct {
	// target API definition, OpenAPI or SOAP, local file or URL
	Target string `yaml:"target"`
	// openapi, soap, or graphql
	Format string `yaml:"format"`
	// the hostname to override in the (remote) OpenAPI spec
	Hostname string `yaml:"hostname"`
	// safe mode this will skip the active scan and perform a baseline scan
	Safe bool `default:"false" yaml:"safe"`
	// config file or URL to use to INFO, IGNORE or FAIL warnings
	Config string `yaml:"config"`
	// show debug messages
	Debug bool `default:"false" yaml:"debug"`
	// short output format - dont show PASSes or example URLs
	Short bool `default:"false" yaml:"short"`
	// minimum level to show: PASS, IGNORE, INFO, WARN or FAIL
	Level string `default:"PASS" yaml:"level"`
	// context file which will be loaded prior to scanning the target
	Context string `yaml:"context"`
	// username to use for authenticated scans - must be defined in the given context file
	User string `yaml:"user"`
	// delay in seconds to wait for passive scanning
	Delay int `yaml:"delay"`
	// max time in minutes to wait for ZAP to start and the passive scan to run
	Time int `default:"0" yaml:"time"`
	// ZAP command line options
	ZapOptions string `yaml:"zap_options"`
	// fail the scan on WARN issues, default true
	FailOnWarn bool `default:"true" yaml:"fail_on_warn"`
}

type BaselineOptions struct {
	// target URL including the protocol
	Target string `yaml:"target"`
	// config file or URL to use to INFO, IGNORE or FAIL warnings
	Config string `yaml:"config"`
	// show debug messages
	Debug bool `default:"false" yaml:"debug"`
	// short output format - dont show PASSes or example URLs
	Short bool `default:"false" yaml:"short"`
	// minimum level to show: PASS, IGNORE, INFO, WARN or FAIL
	Level string `default:"PASS" yaml:"level"`
	// context file which will be loaded prior to scanning the target
	Context string `yaml:"context"`
	// username to use for authenticated scans - must be defined in the given context file
	User string `yaml:"user"`
	// the number of minutes to spider for (default 1)
	Minutes int `default:"1" yaml:"minutes"`
	// delay in seconds to wait for passive scanning
	Delay int `yaml:"delay"`
	// max time in minutes to wait for ZAP to start and the passive scan to run
	Time int `default:"0" yaml:"time"`
	// use the Ajax spider in addition to the traditional one
	Ajax bool `default:"false" yaml:"ajax"`
	// ZAP command line options
	ZapOptions string `yaml:"zap_options"`
	// fail the scan on WARN issues, default true
	FailOnWarn bool `default:"true" yaml:"fail_on_warn"`
}

type FullOptions struct {
	// target URL including the protocol
	Target string `yaml:"target"`
	// config file or URL to use to INFO, IGNORE or FAIL warnings
	Config string `yaml:"config"`
	// show debug messages
	Debug bool `default:"false" yaml:"debug"`
	// short output format - dont show PASSes or example URLs
	Short bool `default:"false" yaml:"short"`
	// minimum level to show: PASS, IGNORE, INFO, WARN or FAIL
	Level string `default:"PASS" yaml:"level"`
	// context file which will be loaded prior to scanning the target
	Context string `yaml:"context"`
	// username to use for authenticated scans - must be defined in the given context file
	User string `yaml:"user"`
	// the number of minutes to spider for (default -1, unlimited)
	Minutes int `default:"-1" yaml:"minutes"`
	// delay in seconds to wait for passive scanning
	Delay int `yaml:"delay"`
	// max time in minutes to wait for ZAP to start and the passive scan to run
	Time int `default:"0" yaml:"time"`
	// use the Ajax spider in addition to the traditional one
	Ajax bool `default:"false" yaml:"ajax"`
	// ZAP command line options
	ZapOptions string `yaml:"zap_options"`
	// fail the scan on WARN issues, default true
	FailOnWarn bool `default:"true" yaml:"fail_on_warn"`
}

type Options struct {
	API      ApiOptions      `yaml:"api"`
	Baseline BaselineOptions `yaml:"baseline"`
	Full     FullOptions     `yaml:"full"`
}

func (a *Options) UnmarshalYAML(yamlFile string) (err error) {
	bytes, err := os.ReadFile(yamlFile)
	if err != nil {
		return err
	}

	if err := defaults.Set(a); err != nil {
		return err
	}

	if err := yaml.Unmarshal(bytes, a); err != nil {
		return err
	}

	return nil
}

func (a *Options) ToFullScanArgs(filename string) (args []string) {
	args = []string{}
	args = appendTargetArg(args, a.Full.Target)
	args = appendConfigArg(args, a.Full.Config)
	args = appendMinutesArg(args, a.Full.Minutes)
	args = appendDebugArg(args, a.Full.Debug)
	args = appendDelayArg(args, a.Full.Delay)
	args = appendFailOnWarnArg(args, a.Full.FailOnWarn)
	args = appendAjaxSpiderArg(args, a.Full.Ajax)
	args = appendLevelArg(args, a.Full.Level)
	args = appendConfigArg(args, a.Full.Context)
	args = appendShortArg(args, a.Full.Short)
	args = appendTimeArg(args, a.Full.Time)
	args = appendUserArg(args, a.Full.User)
	args = appendZapOptionsArg(args, a.Full.ZapOptions)
	args = appendReportArg(args, filename)
	return args
}

func (a *Options) ToBaselineScanArgs(filename string) (args []string) {
	args = []string{}
	args = appendTargetArg(args, a.Baseline.Target)
	args = appendConfigArg(args, a.Baseline.Config)
	args = appendMinutesArg(args, a.Baseline.Minutes)
	args = appendDebugArg(args, a.Baseline.Debug)
	args = appendDelayArg(args, a.Baseline.Delay)
	args = appendFailOnWarnArg(args, a.Baseline.FailOnWarn)
	args = appendAjaxSpiderArg(args, a.Baseline.Ajax)
	args = appendLevelArg(args, a.Baseline.Level)
	args = appendContextArg(args, a.Baseline.Context)
	args = appendShortArg(args, a.Baseline.Short)
	args = appendTimeArg(args, a.Baseline.Time)
	args = appendUserArg(args, a.Baseline.User)
	args = appendZapOptionsArg(args, a.Baseline.ZapOptions)
	args = appendReportArg(args, filename)
	args = append(args, "--auto")
	return args
}

func (a *Options) ToApiScanArgs(filename string) (args []string) {
	args = []string{}
	args = appendTargetArg(args, a.API.Target)
	args = appendFormatArg(args, a.API.Format)
	args = appendConfigArg(args, a.API.Config)
	args = appendDebugArg(args, a.API.Debug)
	args = appendDelayArg(args, a.API.Delay)
	args = appendFailOnWarnArg(args, a.API.FailOnWarn)
	args = appendLevelArg(args, a.API.Level)
	args = appendContextArg(args, a.API.Context)
	args = appendShortArg(args, a.API.Short)
	args = appendSafeArg(args, a.API.Safe)
	args = appendTimeArg(args, a.API.Time)
	args = appendUserArg(args, a.API.User)
	args = appendHostnameArg(args, a.API.Hostname)
	args = appendZapOptionsArg(args, a.API.ZapOptions)
	args = appendReportArg(args, filename)
	return args
}

func appendTargetArg(args []string, target string) []string {
	return appendStringArg(args, "-t", target)
}

func appendFormatArg(args []string, format string) []string {
	return appendStringArg(args, "-f", format)
}

func appendContextArg(args []string, context string) []string {
	return appendStringArg(args, "-n", context)
}

func appendUserArg(args []string, user string) []string {
	return appendStringArg(args, "-U", user)
}

func appendLevelArg(args []string, level string) []string {
	return appendStringArg(args, "-l", level)
}

func appendHostnameArg(args []string, hostname string) []string {
	return appendStringArg(args, "-O", hostname)
}

func appendReportArg(args []string, filename string) []string {
	return appendStringArg(args, "-r", filename)
}

func appendZapOptionsArg(args []string, options string) []string {
	return appendStringArg(args, "-z", options)
}

func appendStringArg(args []string, arg string, value string) []string {
	if len(value) > 0 {
		return append(args, arg, value)
	} else {
		return args
	}
}

func appendConfigArg(args []string, format string) []string {
	if len(format) > 0 {
		if strings.Index(format, "http") == 0 {
			return append(args, "-u", format)
		} else {
			return append(args, "-c", format)
		}
	} else {
		return args
	}
}

func appendDebugArg(args []string, debug bool) []string {
	return appendBoolArg(args, "-d", debug)
}

func appendShortArg(args []string, short bool) []string {
	return appendBoolArg(args, "-s", short)
}

func appendSafeArg(args []string, safe bool) []string {
	return appendBoolArg(args, "-S", safe)
}

func appendFailOnWarnArg(args []string, failOnWarn bool) []string {
	return appendBoolArg(args, "-I", !failOnWarn)
}

func appendAjaxSpiderArg(args []string, ajax bool) []string {
	return appendBoolArg(args, "-j", ajax)
}

func appendBoolArg(args []string, arg string, flag bool) []string {
	if flag {
		return append(args, arg)
	} else {
		return args
	}
}

func appendMinutesArg(args []string, minutes int) []string {
	return appendIntArg(args, "-m", minutes)
}

func appendTimeArg(args []string, time int) []string {
	return appendIntArg(args, "-T", time)
}

func appendDelayArg(args []string, delay int) []string {
	return appendIntArg(args, "-D", delay)
}

func appendIntArg(args []string, arg string, value int) []string {
	if value > 0 {
		return append(args, arg, fmt.Sprint(value))
	} else {
		return args
	}
}
