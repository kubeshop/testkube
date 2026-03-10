package crdcommon

import (
	"bytes"
	"encoding/json"
	"reflect"
	"regexp"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

type SerializeOptions struct {
	OmitCreationTimestamp bool
	CleanMeta             bool
	Kind                  string
	GroupVersion          *schema.GroupVersion
}

type ObjectWithTypeMeta interface {
	SetGroupVersionKind(schema.GroupVersionKind)
}

func AppendTypeMeta(kind string, version schema.GroupVersion, crs ...ObjectWithTypeMeta) {
	for _, cr := range crs {
		cr.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   version.Group,
			Version: version.Version,
			Kind:    kind,
		})
	}
}

func CleanObjectMeta(crs ...metav1.Object) {
	for _, cr := range crs {
		cr.SetGeneration(0)
		cr.SetResourceVersion("")
		cr.SetSelfLink("")
		cr.SetUID("")
		cr.SetFinalizers(nil)
		cr.SetOwnerReferences(nil)
		cr.SetManagedFields(nil)

		annotations := cr.GetAnnotations()
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		cr.SetAnnotations(annotations)
	}
}

var creationTsNullRegex = regexp.MustCompile(`\n\s+creationTimestamp: null`)
var creationTsRegex = regexp.MustCompile(`\n\s+creationTimestamp:[^\n]*`)

// clearYAMLFlowStyle recursively removes flow style from YAML mapping and sequence nodes,
// so they are marshaled as block YAML instead of inline JSON-like format.
// It also clears double-quoted style from scalar nodes so that string keys/values
// from JSON input are serialized as unquoted YAML scalars where possible.
func clearYAMLFlowStyle(node *yaml.Node) {
	switch node.Kind {
	case yaml.MappingNode, yaml.SequenceNode:
		node.Style = 0
	case yaml.ScalarNode:
		if node.Style == yaml.DoubleQuotedStyle || node.Style == yaml.SingleQuotedStyle {
			node.Style = 0
		}
	}
	for _, child := range node.Content {
		clearYAMLFlowStyle(child)
	}
}

func SerializeCRD(cr interface{}, opts SerializeOptions) ([]byte, error) {
	if opts.CleanMeta || (opts.Kind != "" && opts.GroupVersion != nil) {
		// For simplicity, support both direct struct (as in *List.Items), as well as the pointer itself
		if reflect.ValueOf(cr).Kind() == reflect.Struct {
			v := reflect.ValueOf(cr)
			p := reflect.New(v.Type())
			p.Elem().Set(v)
			cr = p.Interface()
		}

		// Deep copy object, as it will have modifications
		if obj, ok := cr.(runtime.Object); ok {
			cr = obj.DeepCopyObject()
		}

		// Clean messy metadata
		if opts.CleanMeta {
			if v, ok := cr.(metav1.Object); ok {
				CleanObjectMeta(v)
				cr = v
			}
		}

		// Append metadata when expected
		if opts.Kind != "" && opts.GroupVersion != nil {
			if v, ok := cr.(ObjectWithTypeMeta); ok {
				AppendTypeMeta(opts.Kind, *opts.GroupVersion, v)
				cr = v
			}
		}
	}

	out, err := json.Marshal(cr)
	if err != nil {
		return nil, err
	}
	var node yaml.Node
	if err := yaml.Unmarshal(out, &node); err != nil {
		return nil, err
	}
	clearYAMLFlowStyle(&node)
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		_ = enc.Encode(node.Content[0])
	} else {
		_ = enc.Encode(&node)
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	b := buf.Bytes()
	if opts.OmitCreationTimestamp {
		b = creationTsRegex.ReplaceAll(b, nil)
	} else {
		b = creationTsNullRegex.ReplaceAll(b, nil)
	}
	return b, err
}

var crdSeparator = []byte("---\n")

// SerializeCRDs builds a serialized version of CRD,
// persisting the order of properties from the struct.
func SerializeCRDs[T interface{}](crs []T, opts SerializeOptions) ([]byte, error) {
	result := []byte(nil)
	for _, cr := range crs {
		b, err := SerializeCRD(cr, opts)
		if err != nil {
			return nil, err
		}
		if len(result) > 0 {
			result = append(append(result, crdSeparator...), b...)
		} else {
			result = b
		}
	}
	return result, nil
}

func DeserializeCRD(cr runtime.Object, content []byte) error {
	_, _, err := scheme.Codecs.UniversalDeserializer().Decode(content, nil, cr)
	return err
}
