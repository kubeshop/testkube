package testkube

import "strings"

// Stop is what the component that ended an execution knows about the stop. The worker writes
// these values into the job annotations, and the control plane sends them with the stop
// transition, so the runner reads one shape for both paths.
type Stop struct {
	// Code is the terminal status that the stop asks for, aborted or canceled.
	Code string
	// Actor is the component that decided the stop, empty when no component decided it.
	Actor StopActor
	// Reason is the code for the cause, empty for a plain cancel.
	Reason StopReason
	// Detail is the free text that the caller sent next to the reason.
	Detail string
	// Causes holds the plain cause of each step that the stop ended, by the reference of the step,
	// with an empty reference for the initialization step. The runner fills it from what it saw,
	// because the heal wraps the message of these steps in the termination sentence. It is nil
	// when the caller knows no more than the step messages.
	Causes map[string]string
}

// Sentence returns the words for the stop, in the brackets of the termination message. It keeps
// the raw reason when the reason has no words, so a code from a newer control plane still reads.
// A stop that names neither an actor nor a reason has no words, and the detail alone says too
// little, so the function returns an empty string.
func (s Stop) Sentence() string {
	if s.Actor == "" && s.Reason == "" {
		return ""
	}
	var parts []string
	if s.Actor != "" {
		parts = append(parts, s.Actor.Sentence())
	}
	if s.Reason != "" {
		words := s.Reason.Sentence()
		if words == "" {
			words = string(s.Reason)
		}
		parts = append(parts, words)
	}
	if s.Detail != "" {
		parts = append(parts, s.Detail)
	}
	return strings.Join(parts, ": ")
}

// cause returns the plain cause of the step. For a step that the stop ended, it is the cause that
// the runner gave. Else it is the message of the step, which the heal did not touch.
func (s Stop) cause(ref string, step TestWorkflowStepResult) string {
	if cause, ok := s.Causes[ref]; ok {
		return cause
	}
	return step.ErrorMessage
}

// StopCauses is what the runner knows about a stop apart from the termination sentence. The heal
// renders the termination message and the plain cause of each step from it, so the two agree.
type StopCauses struct {
	// Stop is the stop as the job reports it.
	Stop Stop
	// Written holds the cause that the runner wrote into a step, by the reference of the step,
	// with an empty reference for the initialization step.
	Written map[string]*Cause
	// Ending is the error that Kubernetes reported in its own words, for example the deadline of the
	// job or the message of a pod failure. It is empty when the error only repeats the stop or a
	// bare container reason such as OOMKilled, because the stop and the code already name them.
	Ending string
}

// written returns the cause that the runner wrote into the step, or nil.
func (c *StopCauses) written(ref string) *Cause {
	if c == nil {
		return nil
	}
	return c.Written[ref]
}

// ending returns the plain words of what ended the execution, for a step with no cause of its own.
// It is the error that Kubernetes reported, else the free text of the stop.
func (c *StopCauses) ending() string {
	if c == nil {
		return ""
	}
	if c.Ending != "" {
		return c.Ending
	}
	return c.Stop.Detail
}

// lead returns the words before a cause that a step holds with its own code. It is the ending, else
// the reason of the stop. A person's stop gives no reason here, because the heading already names
// the cancel.
func (c *StopCauses) lead() string {
	if ending := c.ending(); ending != "" {
		return ending
	}
	if c == nil {
		return ""
	}
	isReasonEmpty := c.Stop.Reason == ""
	isStoppedByPerson := c.Stop.Actor.IsPerson()
	if isReasonEmpty || isStoppedByPerson {
		return ""
	}
	return Cause{Reason: string(c.Stop.Reason), Message: c.Stop.Detail}.String()
}
