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
//
// A toolkit step can also give the step message. The init process sets TK_ERR_FILE to
// constants.StepErrorPath for a toolkit step, and unsets it for the other steps. The file has these rules:
//
//   - The run command removes the file before each attempt starts. When it cannot remove
//     the file, it does not read the file for that attempt.
//   - The toolkit writes the cause of its failure to the file, and then exits with a code
//     that is not zero. The file survives the exit, and it does not go into the log.
//   - When a toolkit step fails, the run command reads the file. It removes the ANSI escape codes,
//     masks the sensitive values, keeps the first line, and limits the size.
//   - The process-killed and step-timeout messages have priority over the file. An aborted step does
//     not read the file, because the abort is the cause. A negative step that passes has no message.
//
// The step message does not go through the obfuscated log stream, so the run command masks each
// sensitive value of the file, also a short value. It masks before it cuts the text, because a part
// of a sensitive value does not match the masking.
package orchestration
