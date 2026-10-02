// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

// Package testworkflowresolver applies the templates and the config of a test workflow, so the
// workflow holds its final steps and services.
//
// # Error messages
//
// An error of this package becomes the message of an execution that could not start:
//
//   - The error names the path in the workflow and the template that it uses, for example
//     `spec.steps[0].template "x": <cause>`. The paths join through their error type, so a path of
//     the workflow and a path inside the template read as one path.
//   - A config value that an expression reads and that nothing sets gives `the config value
//     "<name>" is not set`. That is what the user fixes, so the error does not name the path of the
//     expression inside the template.
//   - A template that does not exist gives `the template does not exist` inside the path that names it.
package testworkflowresolver
