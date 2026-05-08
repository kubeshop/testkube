package testtriggers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestMapResourceFieldRefAPIToKube_InvalidDivisorDefaultsToOne(t *testing.T) {
	mapped := mapResourceFieldRefAPIToKube(&testkube.ResourceFieldRef{
		ContainerName: "app",
		Resource:      "limits.cpu",
		Divisor:       "not-a-quantity",
	})

	assert.Equal(t, "1", mapped.Divisor.String())
}

func TestMapResourceFieldRefAPIToKube_ValidDivisorIsPreserved(t *testing.T) {
	mapped := mapResourceFieldRefAPIToKube(&testkube.ResourceFieldRef{
		ContainerName: "app",
		Resource:      "limits.cpu",
		Divisor:       "1m",
	})

	assert.Equal(t, "1m", mapped.Divisor.String())
}
