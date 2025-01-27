package uicrd

import (
	"os"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeshop/testkube/internal/crdcommon"
	"github.com/kubeshop/testkube/pkg/ui"
)

func PrintCRD[T interface{}](cr T, kind string, groupVersion schema.GroupVersion) {
	PrintCRDs([]T{cr}, kind, groupVersion)
}

func PrintCRDs[T interface{}](crs []T, kind string, groupVersion schema.GroupVersion) {
	bytes, err := crdcommon.SerializeCRDs(crs, crdcommon.SerializeOptions{
		OmitCreationTimestamp: true,
		CleanMeta:             true,
		Kind:                  kind,
		GroupVersion:          &groupVersion,
	})
	ui.ExitOnError("serializing the crds", err)
	_, _ = os.Stdout.Write(bytes)
}
