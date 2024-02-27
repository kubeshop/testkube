package common

import (
	"encoding/json"
	"reflect"
	"regexp"

	"gopkg.in/yaml.v2"
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
		switch cr.(type) {
		case runtime.Object:
			cr = cr.(runtime.Object).DeepCopyObject()
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
	m := yaml.MapSlice{}
	_ = yaml.Unmarshal(out, &m)
	b, _ := yaml.Marshal(m)
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
