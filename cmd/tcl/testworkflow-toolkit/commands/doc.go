// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

// Package commands holds the commercial testworkflow-toolkit commands.
//
// # Services cleanup retries
//
// kill.go runs the services cleanup of the kill step - worker.List, each getServiceHealth pod
// read, and worker.DestroyGroup - through the retryTransient helper and the classifiers in
// retry.go, bounded by cleanupRetryCount and cleanupRetryDelay. Only a failure whose every branch
// is transient is downgraded to a warning: the test steps already have their result, so a
// teardown blip is not the test's verdict. Anything else still fails the step. retry.go documents
// which error classes count as transient and why.
//
// # Agent step
//
// agent.go runs the `agent` step: one turn of an AI Agent that runs in the control plane. The
// command starts the turn, reads it until its end, prints its tool calls and text to the step
// log, writes the answer to a file, and writes the step output `session`. It writes `session`
// when the control plane first gives the session ID, so a failed turn has it too.
//
//   - The processor gives the spec as base64 JSON, so the command resolves the expressions of
//     the name and the prompt. It resolves them one time: an expression in the value of a step
//     output stays text. The prompt reads what a step of the init process reads, the data of
//     the executions that an earlier step started included.
//   - The command fails before the first call when the control plane does not publish the
//     capability of the step.
//   - Text from the AI Agent and its tools loses each byte 0x01 before the command prints it or
//     writes it. With that byte the text can start an instruction for the init process.
//   - The answer is the file response.md in the results directory of the step, and it is not a
//     step output. The init process skips an output file above data.MaxOutputSize, and an
//     answer has no size limit. A step with no ID has no results directory, so the command
//     fails before it starts a turn.
//   - The command makes one request ID for the process and sends it with the start call, so a
//     retry of that call returns the same turn.
//   - When the step stops before the turn ends, the command asks the control plane one time to
//     cancel the turn. The call has its own short deadline, because the context of the step is
//     done by then.
//   - The command does not mask the prompt. It cannot read the list of secret values.
package commands
