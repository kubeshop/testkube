// Package testkube holds the API model of Testkube and the behaviour that belongs to it.
//
// Most files in this package are generated from the OpenAPI specification in api/v1/testkube.yaml.
// A file that ends with _extended holds the hand-written methods of the model next to it. Do not
// edit a generated file, because the next run of the generator reverts the change.
//
// # Why an execution did not pass
//
// A result that is not passed carries a TestWorkflowStatusDetails object. The object answers four
// questions: which layer failed, what the cause was, where it happened, and who decided a stop.
//
//   - Type is the layer: init-failure, execution-failure, step-failure, user-cancel, or unknown.
//     The dashboard reads it for the label, and a filter and a webhook condition select on it.
//   - Reason is a code for the cause. The codes live in stop_reason.go and start_reason.go, each
//     with a sentence for a message. The values are stable, because filters and telemetry use them.
//   - Message and Step name the step that holds the cause. They stay inside the deployment.
//     The message holds only the cause, because the type, the reason and the status already have
//     their own place. The step message keeps the full termination sentence. So the runner gives
//     the classifier the plain cause of each step that a stop ended, from the values it saw.
//   - Actor is the component that decided a stop. The codes live in stop_actor.go.
//   - User is the person who canceled. The control plane fills it, and the runner never does.
//
// StatusDetailsTypeOf maps a reason code to its layer. An actor that is a person always gives
// user-cancel, because a cancel is not a failure. NewStatusDetails builds the object for a stop.
//
// # Display labels
//
// Label gives the raw type and reason, for tables such as the CLI. StatusDetailsType.DisplayName
// gives the display name of a type, and DisplayLabel adds the reason code to it, for a short
// summary such as a CI check. Both hold only the type and the reason. A new type needs a display
// name in statusDetailsTypeNames, else it shows as its code.
//
// # The stop
//
// Stop carries what the component that ended an execution knows, and its own documentation holds
// the fields and where they come from. Stop.Sentence renders the words that the result stores.
//
// HealAbortedOrCanceled writes the code of a stop into the step that it stops, unless that step
// already holds a code of its own.
//
// # The classifier
//
// ClassifyStatus builds the object from the final result and from the stop. The caller runs it
// after the heal functions, because the object reads the statuses that they settle. Its own
// documentation holds the order of the rules and the reason for that order.
//
// The runner classifies when a watch ends, after a lost watch, and after the check for stuck
// executions. The standalone agent classifies when it declines an execution. The control plane
// writes the object for the stops that only it decides, and it never writes one for a running
// execution, because the next save of the runner replaces every result field.
//
// # Execution lineage and reruns
//
// TestWorkflowExecutionLineage (baseId, rootId, attempt) records what an execution is a rerun of.
// It is written for every execution, not only reruns: an original run is its own root at attempt
// 1, so "every execution of chain R" is one predicate and includes the original. In SQL that is
// COALESCE(lineage_root_id, id) = R, matching the chain index, because a legacy row carries NULL
// there and is its own root. TestWorkflowExecution.EffectiveLineage synthesizes that default for
// rows written before the columns existed, which is why nothing has to be backfilled. Derive it
// there and nowhere else, or the two repositories will disagree.
//
// The lineage travels ScheduleRequest.base_execution_id -> Enqueuer.deriveLineage -> the
// execution record -> ExecutionStart.lineage -> ExecutionConfig.Lineage -> RerunExecutionId in
// the pod, which is what makes the reserved execution("rerun") reference resolve. Every path that
// builds an ExecutionStart must set it from the execution record, in this repository and in
// testkube-cloud-api: a writer that forgets sends the pod no lineage, execution("rerun") silently
// stops resolving, and nothing reports it. That has already happened once. `go build` cannot
// catch it, because an unused helper function is not a compile error; the unused linter is the
// guard.
//
// Only the base ID travels on the wire. The Control Plane derives the root and attempt by loading
// the base through the environment-scoped results repository, which is also where it confirms
// the base belongs to the caller's environment: proto/service.proto requires that, and the load
// is what enforces it. A caller able to assert a root or an attempt could forge a chain.
//
// An original run has an empty baseId and must resolve to no rerun. Returning rootId there would
// make every execution a rerun of itself.
//
// Reserved references win over the registry. execution("parent") and execution("rerun") are
// resolved before the executions the workflow scheduled, so a child aliased - or a workflow named
// - parent or rerun cannot shadow them. A collision is refused rather than resolved either way,
// because preferring the reserved meaning would instead make that child unreachable by name.
// executiondata.IsReservedRef is the list.
package testkube
