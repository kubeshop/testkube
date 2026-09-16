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
// No code in this package writes the object yet. The runner and the control plane fill it, and
// each of them records where it does so.
package testkube
