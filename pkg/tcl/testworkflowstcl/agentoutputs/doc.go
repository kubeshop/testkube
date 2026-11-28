// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

// Package agentoutputs holds the rules for the outputs that an agent step declares, which do not
// need the answer of the AI Agent.
//
// The processor applies them when it processes the workflow, so a wrong declaration fails before
// a pod starts. The toolkit applies them again in the pod, because a Runner can be older than
// the control plane that processed the workflow. The toolkit also holds the rules that need the
// answer: cmd/tcl/testworkflow-toolkit/commands/agent_outputs.go.
package agentoutputs
