package specs

import (
	"bytes"
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"

	"github.com/kubeshop/testkube/pkg/executor/options"
	"github.com/kubeshop/testkube/pkg/utils"
)

// NewPersistentVolumeClaimSpec is a method to create new persistent volume claim spec
func NewPersistentVolumeClaimSpec(log *zap.SugaredLogger, options options.JobOptions) (*corev1.PersistentVolumeClaim, error) {
	tmpl, err := utils.NewTemplate("volume-claim").Parse(options.PvcTemplate)
	if err != nil {
		return nil, fmt.Errorf("creating volume claim spec from pvc template error: %w", err)
	}

	var buffer bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buffer, "volume-claim", options); err != nil {
		return nil, fmt.Errorf("executing volume claim spec pvc template: %w", err)
	}

	var pvc corev1.PersistentVolumeClaim
	pvcSpec := buffer.String()
	if options.PvcTemplateExtensions != "" {
		tmplExt, err := utils.NewTemplate("jobExt").Parse(options.PvcTemplateExtensions)
		if err != nil {
			return nil, fmt.Errorf("creating pvc extensions spec from executor template error: %w", err)
		}

		var bufferExt bytes.Buffer
		if err = tmplExt.ExecuteTemplate(&bufferExt, "jobExt", options); err != nil {
			return nil, fmt.Errorf("executing pvc extensions spec executor template: %w", err)
		}

		if pvcSpec, err = merge2.MergeStrings(bufferExt.String(), pvcSpec, false, kyaml.MergeOptions{}); err != nil {
			return nil, fmt.Errorf("merging spvc spec executor templates: %w", err)
		}
	}

	log.Debug("Volume claim specification", pvcSpec)
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(pvcSpec), len(pvcSpec))
	if err := decoder.Decode(&pvc); err != nil {
		return nil, fmt.Errorf("decoding pvc spec error: %w", err)
	}

	return &pvc, nil
}
