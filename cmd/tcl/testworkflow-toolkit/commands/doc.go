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
// log, and writes the answer to a file. It also writes the file agent.json in the results
// directory of the step, with the session and the name of the AI Agent. The init process reads
// it for step.<id>.agent.session and step.<id>.agent.name. The command writes it when the
// control plane first gives the session ID, so a failed turn has it too. The session is not a
// step output: the outputs of all steps make one set of names for the workflow, so the session
// of a second agent step would replace the first.
//
//   - The processor gives the spec as base64 JSON, so the command resolves the expressions of
//     the name and the prompt. It resolves them one time: an expression in the value of a step
//     output stays text. The prompt reads the same values as a step of the init process, and
//     also the data of the executions that an earlier step started.
//   - The command fails before the first call when the control plane does not publish the
//     capability of the step.
//   - Text from the AI Agent and its tools loses each byte 0x01 before the command prints it or
//     writes it. With that byte the text can start an instruction for the init process.
//   - The answer is the file response.md in the results directory of the step, and it is not a
//     step output. The init process skips an output file above data.MaxOutputSize, and an
//     answer has no size limit. A later step reads the path of the file at
//     step.<id>.agent.responseFile, so the author sets the id or the name of the step.
//   - agent_outputs.go gives the outputs that the step declares under `outputs`. Each one has
//     the schema of a config parameter. The command adds a contract to the prompt: the answer
//     ends with a fenced block with the language testkube-outputs that holds one JSON object.
//     The command reads the last block, checks each field against its schema, and writes each
//     value as a step output, so a later step reads step.<id>.outputs.<name>. A value is text,
//     a number or a boolean; a list and an object fail. A field that the answer does not have
//     takes its default.
//   - A control plane with the capability tw-agent-step-outputs gives the AI Agent a tool for the
//     outputs. The command sends it the schemas, adds no contract to the prompt, and takes the
//     fields that the turn returns. It checks them the same way, because it trusts no other check.
//   - An answer with no valid outputs fails the step. Then, and when the turn fails or the
//     control plane refuses it, the command writes the default of each output that has one. A
//     condition on an output that does not exist aborts the execution, and with a default it
//     reads a known value, such as "no-go". An output with no default has no value then.
//   - The command fails before the first call for an output name that the init process does
//     not read as a file, and for a default that its schema refuses. The processor refuses
//     the same names when it processes the workflow.
//   - The processor gives an ID to each step, so each agent step has a results directory. When
//     the step has no results directory, the command fails before it starts a turn.
//   - The ID goes to the first stage of a step only, and the init process collects the outputs
//     from the stage with the ID. So the processor refuses an agent step with an
//     operation that makes an earlier stage, such as a delay.
//   - The command makes one request ID for the process and sends it with the start call, so a
//     retry of that call returns the same turn.
//   - When the step stops before the turn ends, the command asks the control plane one time to
//     cancel the turn. The call has its own short deadline, because the context of the step is
//     done by then.
//   - The command does not mask the prompt. It cannot read the list of secret values.
package commands
