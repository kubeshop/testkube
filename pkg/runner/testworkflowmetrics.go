package runner

import (
	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
)

type stepStats struct {
	numSteps      int
	numExecute    int
	hasArtifacts  bool
	hasMatrix     bool
	hasParallel   bool
	hasTemplate   bool
	hasServices   bool
	imagesUsed    map[string]struct{}
	templatesUsed map[string]struct{}
}

func (ss *stepStats) Merge(stats *stepStats) {
	ss.numSteps += stats.numSteps
	ss.numExecute += stats.numExecute

	if stats.hasArtifacts {
		ss.hasArtifacts = true
	}
	if stats.hasMatrix {
		ss.hasMatrix = true
	}
	if stats.hasParallel {
		ss.hasParallel = true
	}
	if stats.hasServices {
		ss.hasServices = true
	}
	if stats.hasTemplate {
		ss.hasTemplate = true
	}
	for image := range stats.imagesUsed {
		ss.imagesUsed[image] = struct{}{}
	}
	for tmpl := range stats.templatesUsed {
		ss.templatesUsed[tmpl] = struct{}{}
	}
}

func getStepInfo(step testworkflowsv1.Step) *stepStats {
	res := &stepStats{
		imagesUsed:    make(map[string]struct{}),
		templatesUsed: make(map[string]struct{}),
	}
	if step.Execute != nil {
		res.numExecute++
	}
	if step.Artifacts != nil {
		res.hasArtifacts = true
	}
	if len(step.Use) > 0 {
		res.hasTemplate = true
		for _, tmpl := range step.Use {
			res.templatesUsed[tmpl.Name] = struct{}{}
		}
	}
	if step.Template != nil {
		res.hasTemplate = true
		res.templatesUsed[step.Template.Name] = struct{}{}
	}
	if len(step.Services) > 0 {
		res.hasServices = true
	}

	if step.Run != nil && step.Run.Image != "" {
		res.imagesUsed[step.Run.Image] = struct{}{}
	}
	if step.Container != nil && step.Container.Image != "" {
		res.imagesUsed[step.Container.Image] = struct{}{}
	}

	for _, step := range step.Steps {
		res.Merge(getStepInfo(step))
	}

	if step.Parallel != nil {
		res.hasParallel = true

		if len(step.Parallel.Matrix) != 0 {
			res.hasMatrix = true
		}
		if step.Parallel.Artifacts != nil {
			res.hasArtifacts = true
		}
		if step.Parallel.Execute != nil {
			res.numExecute++
		}
		if len(step.Parallel.Use) > 0 {
			res.hasTemplate = true
			for _, tmpl := range step.Parallel.Use {
				res.templatesUsed[tmpl.Name] = struct{}{}
			}
		}
		if step.Parallel.Template != nil {
			res.hasTemplate = true
			res.templatesUsed[step.Parallel.Template.Name] = struct{}{}
		}

		if len(step.Parallel.Services) > 0 {
			res.hasServices = true
		}

		if step.Parallel.Run != nil && step.Parallel.Run.Image != "" {
			res.imagesUsed[step.Parallel.Run.Image] = struct{}{}
		}
		if step.Parallel.Container != nil && step.Parallel.Container.Image != "" {
			res.imagesUsed[step.Parallel.Container.Image] = struct{}{}
		}

		for _, step := range step.Parallel.Steps {
			res.Merge(getStepInfo(step))
		}
	}

	return res
}
