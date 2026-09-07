package action

import (
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

// buildTestCasesAction flattens the CRD policy into the shape the init process
// carries. Nil in, nil out: a step without the block costs nothing.
//
// This lives in the open source package for the same reason the verdict does.
// The policy can only reach here from a Pro preset, which registers
// ProcessTestCases where the open source preset registers StubTestCases, so an
// open source deployment never produces one to flatten.
func buildTestCasesAction(policy *testworkflowsv1.StepTestCases) *lite.ActionTestCases {
	if policy == nil {
		return nil
	}

	action := &lite.ActionTestCases{Enforce: policy.Enforce}

	if policy.Report != nil {
		action.ReportPaths = policy.Report.Paths
		action.ReportFormat = policy.Report.Format
		action.OnMissing = policy.Report.OnMissing
	}

	if policy.Mute != nil {
		action.MuteInclude = policy.Mute.Include
		action.MuteExclude = policy.Mute.Exclude
	}

	if policy.Tolerate != nil {
		action.Tolerate = &lite.ActionTestCasesTolerance{
			MaxFailed:        policy.Tolerate.MaxFailed,
			MaxFailedPercent: policy.Tolerate.MaxFailedPercent,
			MinPassed:        policy.Tolerate.MinPassed,
			MinPassedPercent: policy.Tolerate.MinPassedPercent,
		}
	}

	if policy.Select != nil {
		action.Select = &lite.ActionTestCasesSelect{
			From:         policy.Select.From,
			Status:       policy.Select.Status,
			Include:      policy.Select.Include,
			Exclude:      policy.Select.Exclude,
			Cases:        policy.Select.Cases,
			IncludeMuted: policy.Select.IncludeMuted,
			As:           policy.Select.As,
			Collapse:     policy.Select.Collapse,
			Empty:        policy.Select.Empty,
		}
		if policy.Select.Write != nil {
			action.Select.WritePath = policy.Select.Write.Path
			action.Select.WriteSep = policy.Select.Write.Separator
		}
	}

	return action
}
