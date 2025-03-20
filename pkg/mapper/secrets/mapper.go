package secrets

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

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
	if len(secret.OwnerReferences) > 0 {
		owner = &testkube.SecretOwner{
			Kind: secret.OwnerReferences[0].Kind,
			Name: secret.OwnerReferences[0].Name,
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

	secretType := string(secret.Type)
	if secret.Type == corev1.SecretTypeOpaque {
		secretType = ""
	}

	updateTime := secret.CreationTimestamp.Time
	for _, field := range secret.ManagedFields {
		if field.Time != nil && field.Time.After(updateTime) {
			updateTime = field.Time.Time
		}
	}

	return testkube.Secret{
		Name:       secret.Name,
		Namespace:  secret.Namespace,
		CreatedAt:  secret.CreationTimestamp.Time,
		UpdatedAt:  updateTime,
		Type_:      secretType,
		Labels:     secret.Labels,
		Controlled: controlled,
		Owner:      owner,
		Keys:       keys,
	}
}
