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
//   - Actor is the component that decided a stop. The codes live in stop_actor.go.
//   - User is the person who canceled. The control plane fills it, and the runner never does.
//
// StatusDetailsTypeOf maps a reason code to its layer. An actor that is a person always gives
// user-cancel, because a cancel is not a failure. NewStatusDetails builds the object for a stop.
//
// # Words for people
//
// Label gives the raw type and reason, for tables such as the CLI. StatusDetailsType.DisplayName
// gives the words that the dashboard shows for a type, and DisplayLabel adds the reason code to
// them, for text that leaves the product, such as a GitHub check. Neither one carries the message
// or the user. A new type needs words in statusDetailsTypeNames and in the dashboard constants, or
// it reads as its code.
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
package testkube
