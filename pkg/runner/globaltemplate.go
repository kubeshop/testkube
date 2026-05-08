package runner

import (
	"context"
	"strings"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/crdcommon"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
)

type GlobalTemplateFactory func(environmentId string) (*testworkflowsv1.TestWorkflowTemplate, error)

func GlobalTemplateInline(yaml string) GlobalTemplateFactory {
	var globalTemplateInline *testworkflowsv1.TestWorkflowTemplate
	if yaml != "" {
		globalTemplateInline = new(testworkflowsv1.TestWorkflowTemplate)
		err := crdcommon.DeserializeCRD(globalTemplateInline, []byte("spec:\n  "+strings.ReplaceAll(yaml, "\n", "\n  ")))
		globalTemplateInline.Name = inlinedGlobalTemplateName
		if err != nil {
			log.DefaultLogger.Errorw("failed to unmarshal inlined global template", "error", err)
			globalTemplateInline = nil
		}
	}
	return func(_ string) (*testworkflowsv1.TestWorkflowTemplate, error) {
		return globalTemplateInline, nil
	}
}

func GlobalTemplateSourced(client testworkflowtemplateclient.TestWorkflowTemplateClient, name string) GlobalTemplateFactory {
	return func(environmentId string) (*testworkflowsv1.TestWorkflowTemplate, error) {
		tpl, err := client.Get(context.Background(), environmentId, name)
		if err != nil {
			return nil, err
		}
		return testworkflows.MapTemplateAPIToKube(tpl), nil
	}
}
