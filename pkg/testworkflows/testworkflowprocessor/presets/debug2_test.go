package presets

import (
"context"
"encoding/json"
"fmt"
"testing"

corev1 "k8s.io/api/core/v1"

testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
"github.com/kubeshop/testkube/internal/common"
"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"

"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/action/actiontypes/lite"
)

func TestDebugActionContainerConfigs(t *testing.T) {
wf := &testworkflowsv1.TestWorkflow{
Spec: testworkflowsv1.TestWorkflowSpec{
TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
Content: &testworkflowsv1.Content{
Git: &testworkflowsv1.ContentGit{
Uri:      "https://github.com/example/example",
Revision: "main",
Paths:    []string{"tests/"},
},
},
Container: &testworkflowsv1.ContainerConfig{
Image:      "microsoft/playwright:v1.44.0-jammy",
WorkingDir: common.Ptr("/data/repo"),
SecurityContext: &corev1.SecurityContext{
RunAsNonRoot:             common.Ptr(true),
AllowPrivilegeEscalation: common.Ptr(false),
Capabilities: &corev1.Capabilities{
Drop: []corev1.Capability{"ALL"},
},
SeccompProfile: &corev1.SeccompProfile{
Type: corev1.SeccompProfileTypeRuntimeDefault,
},
},
},
},
Steps: []testworkflowsv1.Step{
{
StepMeta: testworkflowsv1.StepMeta{
Name: "Install dependencies",
Pure: common.Ptr(true),
},
StepOperations: testworkflowsv1.StepOperations{
Run: &testworkflowsv1.StepRun{
Shell: common.Ptr("npm ci"),
},
},
},
},
},
}

res, err := proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig, ScheduledAt: dummyTime})
if err != nil {
t.Fatalf("Bundle error: %v", err)
}

// Get the actions from the annotations
actions := res.Actions()
fmt.Printf("Number of action groups: %d\n", len(actions))
for i, group := range actions {
fmt.Printf("\n=== Action Group %d (%d actions) ===\n", i, len(group))
for j, a := range group {
if a.Container != nil {
fmt.Printf("  Action[%d]: Container ref=%s\n", j, a.Container.Ref)
sc := a.Container.Config.SecurityContext
if sc != nil {
b, _ := json.MarshalIndent(sc, "    ", "  ")
fmt.Printf("    SecurityContext: %s\n", string(b))
} else {
fmt.Printf("    SecurityContext: nil\n")
}
} else if a.Execute != nil {
fmt.Printf("  Action[%d]: Execute ref=%s toolkit=%v pure=%v\n", j, a.Execute.Ref, a.Execute.Toolkit, a.Execute.Pure)
} else if a.Setup != nil {
fmt.Printf("  Action[%d]: Setup copyInit=%v copyToolkit=%v\n", j, a.Setup.CopyInit, a.Setup.CopyToolkit)
}
}
}

// Now let's look at the LiteActions
liteActions := res.LiteActions()
fmt.Printf("\nNumber of lite action groups: %d\n", len(liteActions))
for i, group := range liteActions {
fmt.Printf("\n=== Lite Action Group %d (%d actions) ===\n", i, len(group))
for j, a := range group {
if a.Type() == lite.ActionTypeContainerTransition {
fmt.Printf("  LiteAction[%d]: Container\n", j)
} else if a.Type() == lite.ActionTypeExecute {
fmt.Printf("  LiteAction[%d]: Execute ref=%s\n", j, a.Execute.Ref)
}
}
}

// Just to show the full spec with actual SecurityContext
spec := res.Job.Spec.Template.Spec
for _, c := range spec.InitContainers {
if c.SecurityContext != nil && c.SecurityContext.Capabilities != nil {
fmt.Printf("\nInit Container %s: capabilities.drop=%v\n", c.Name, c.SecurityContext.Capabilities.Drop)
}
}
for _, c := range spec.Containers {
if c.SecurityContext != nil && c.SecurityContext.Capabilities != nil {
fmt.Printf("Container %s: capabilities.drop=%v\n", c.Name, c.SecurityContext.Capabilities.Drop)
}
}
}
