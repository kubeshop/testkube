package secrets

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapSecretOwnerAPIToKube(owner *testkube.SecretOwner) string {
	if owner != nil && owner.Kind != nil && *owner.Kind != "" && owner.Name != "" {
		return fmt.Sprintf("%s/%s", *owner.Kind, owner.Name)
	}
	return ""
}

func MapSecretKubeToAPI(secret *corev1.Secret) testkube.Secret {
	// Fetch the available keys
	keys := make([]string, 0, len(secret.Data)+len(secret.StringData))
	for k := range secret.Data {
		keys = append(keys, k)
	}
	for k := range secret.StringData {
		keys = append(keys, k)
	}

	// Fetch ownership details
	var owner *testkube.SecretOwner
	kind, name, _ := strings.Cut(secret.Labels["testkubeOwner"], "/")
	if kind != "" && name != "" {
		owner = &testkube.SecretOwner{
			Kind: common.Ptr(testkube.SecretOwnerKind(kind)),
			Name: name,
		}
	}

	// Ensure it's not created externally
	controlled := secret.Labels["createdBy"] == "testkube"

	// Clean up the labels
	delete(secret.Labels, "createdBy")
	delete(secret.Labels, "testkubeOwner")
	if len(secret.Labels) == 0 {
		secret.Labels = nil
	}

	return testkube.Secret{
		Name:       secret.Name,
		Labels:     secret.Labels,
		Controlled: controlled,
		Owner:      owner,
		Keys:       keys,
	}
}
