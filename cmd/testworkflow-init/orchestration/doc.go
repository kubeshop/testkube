// Package orchestration runs the step processes inside the init process and sends the
// state of each step to the watcher as hint instructions.
//
// # Step messages
//
// FinishExecution sends the execution result of a step. The Details field of that result
// becomes the step message of the stored execution. The init process sets Details only for
// the stops that it detects:
//
//   - process-killed: SIGKILL from outside the init process stops the step process, for
//     example the out-of-memory killer. Another signal sets no details, because a stop of the
//     pod has its own cause. When the init process aborts the group itself, it sets no details.
//   - step-timeout: the timeout of the step or of a parent group ends, also before the step starts.
//
// A kill of the whole container also stops the init process, so the init process sends no step message.
package orchestration
