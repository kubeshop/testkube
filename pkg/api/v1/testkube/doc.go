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
// user-cancel, because a cancel is not a failure.
//
// # The stop
//
// Stop is what the component that ended an execution knows: the terminal status it asks for, the
// actor, the reason, and a free-text detail. The worker writes these values into the annotations of
// the job, and the control plane sends them with the stop transition, so the runner reads one shape
// for both paths. Stop.Sentence renders the words that the result stores for the stop.
//
// # The classifier
//
// ClassifyStatus builds the object from the final result and from the stop. The caller runs it
// after the heal functions, because the object reads the statuses that they settle. The first rule
// that matches wins, and status_classify.go carries the reason for each order:
//
//  1. A person decided the stop, so the result is a cancel.
//  2. A step holds a cause of its own, for example a signal of Kubernetes or a code of a toolkit
//     step. The cause wins over the stop, because the cause is what the user fixes.
//  3. The stop names a reason, or an actor that stops without one.
//  4. A step of the test failed.
//  5. A caller that does not name itself removed the job.
//  6. No signal explains the result.
//
// The runner classifies when a watch ends, after a lost watch, and after the check for stuck
// executions. The standalone agent classifies when it declines an execution. The control plane
// writes the object for the stops that only it decides, and it never writes one for a running
// execution, because the next save of the runner replaces every result field.
package testkube
