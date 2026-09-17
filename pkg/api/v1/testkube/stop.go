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
