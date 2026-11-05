package declarative

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	insightsv1 "github.com/kubeshop/testkube/api/insights/v1alpha1"
)

// lastAppliedAnnotation is kubectl's copy of the applied object, which a
// custom resource exported from a cluster carries and the Control Plane has no
// use for.
const lastAppliedAnnotation = "kubectl.kubernetes.io/last-applied-configuration"

// listKind is the kind of the generic list Encode writes for several objects
// in JSON, as kubectl does.
const listKind = "List"

// Decode reads every object of a multi-document YAML (or JSON) stream. Empty
// documents are skipped, and a v1 List is unpacked into its items. Fields the
// schema does not know and duplicate keys are an error, so a typo is reported
// rather than silently dropped. The objects are not validated: see Normalize.
func Decode(r io.Reader) ([]Object, error) {
	reader := utilyaml.NewYAMLReader(bufio.NewReader(r))
	var objects []Object
	for doc := 1; ; doc++ {
		data, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return objects, nil
		}
		if err != nil {
			return nil, fmt.Errorf("document %d: %w", doc, err)
		}
		decoded, err := decodeDocument(data)
		if err != nil {
			return nil, fmt.Errorf("document %d: %w", doc, err)
		}
		objects = append(objects, decoded...)
	}
}

func decodeDocument(data []byte) ([]Object, error) {
	// Strict, so a duplicate key is an error: the lenient conversion keeps
	// only the last of two spec blocks, and applying that would drop the
	// reports of the first.
	data, err := yaml.YAMLToJSONStrict(data)
	if err != nil {
		return nil, err
	}
	if trimmed := strings.TrimSpace(string(data)); trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	var meta metav1.TypeMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	if meta.APIVersion == "v1" && meta.Kind == listKind {
		var list struct {
			Items []json.RawMessage `json:"items"`
		}
		if err := json.Unmarshal(data, &list); err != nil {
			return nil, err
		}
		var objects []Object
		for i, item := range list.Items {
			obj, err := DecodeObject(item)
			if err != nil {
				return nil, fmt.Errorf("items[%d]: %w", i, err)
			}
			objects = append(objects, obj)
		}
		return objects, nil
	}
	obj, err := DecodeObject(data)
	if err != nil {
		return nil, err
	}
	return []Object{obj}, nil
}

// DecodeObject decodes one object, in YAML or JSON, dispatching on its
// apiVersion and kind.
func DecodeObject(data []byte) (Object, error) {
	var meta metav1.TypeMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	if meta.APIVersion == "" || meta.Kind == "" {
		return nil, fmt.Errorf("apiVersion and kind are required")
	}
	kind, err := LookupKind(meta.APIVersion, meta.Kind)
	if err != nil {
		return nil, err
	}
	obj := kind.New()
	if err := yaml.UnmarshalStrict(data, obj); err != nil {
		return nil, fmt.Errorf("%s: %w", kind.Kind, err)
	}
	return obj, nil
}

// PrepareForApply returns a copy of the object holding only what a PUT
// sends: apiVersion, kind, name, labels, annotations and spec. Status and the
// metadata a cluster manages are dropped, so an object exported from either
// the Control Plane or a cluster can be applied as it is.
func PrepareForApply(obj Object) (Object, error) {
	kind, err := KindOf(obj)
	if err != nil {
		return nil, err
	}
	out := obj.DeepCopyObject().(Object)
	out.GetObjectKind().SetGroupVersionKind(kind.GroupVersionKind())
	meta := metav1.ObjectMeta{
		Name:        obj.GetName(),
		Labels:      obj.GetLabels(),
		Annotations: obj.GetAnnotations(),
	}
	if _, ok := meta.Annotations[lastAppliedAnnotation]; ok {
		annotations := make(map[string]string, len(meta.Annotations))
		for k, v := range meta.Annotations {
			if k != lastAppliedAnnotation {
				annotations[k] = v
			}
		}
		meta.Annotations = annotations
	}
	switch o := out.(type) {
	case *insightsv1.Board:
		o.ObjectMeta = meta
		o.Status = commonv1.ControlPlaneResourceStatus{}
	}
	return out, nil
}

// ApplyPayload returns the JSON body of a PUT for the object: the object as
// PrepareForApply leaves it, with no status at all.
func ApplyPayload(obj Object) ([]byte, error) {
	prepared, err := PrepareForApply(obj)
	if err != nil {
		return nil, err
	}
	cleaned, err := clean(prepared)
	if err != nil {
		return nil, err
	}
	return json.Marshal(cleaned)
}

// Encode writes objects as YAML (one document each, so the output can be
// applied again) or JSON (the object itself, or a v1 List of several).
func Encode(objects []Object, format string) ([]byte, error) {
	cleaned := make([]any, 0, len(objects))
	for _, obj := range objects {
		c, err := clean(obj)
		if err != nil {
			return nil, err
		}
		cleaned = append(cleaned, c)
	}
	switch format {
	case "yaml":
		var buf bytes.Buffer
		for i, obj := range cleaned {
			data, err := yaml.Marshal(obj)
			if err != nil {
				return nil, err
			}
			if i > 0 {
				buf.WriteString("---\n")
			}
			buf.Write(data)
		}
		return buf.Bytes(), nil
	case "json":
		var value any = map[string]any{"apiVersion": "v1", "kind": listKind, "items": cleaned}
		if len(cleaned) == 1 {
			value = cleaned[0]
		}
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	}
	return nil, fmt.Errorf("unsupported output format %q (use yaml or json)", format)
}

// clean drops what an object always serializes even when unset - a null
// creationTimestamp and an empty status - so an export reads like the file
// it was applied from.
func clean(obj Object) (map[string]any, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if meta, ok := out["metadata"].(map[string]any); ok {
		if v, set := meta["creationTimestamp"]; set && v == nil {
			delete(meta, "creationTimestamp")
		}
	}
	if status, ok := out["status"].(map[string]any); ok && len(status) == 0 {
		delete(out, "status")
	}
	return out, nil
}
