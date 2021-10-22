package detector

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/stretchr/testify/assert"
)

func TestDetector(t *testing.T) {

	t.Run("detect postman/collection", func(t *testing.T) {

		detector := Detector{}
		detector.Add(PostmanCollectionAdapter{})

		name := detector.Detect(client.CreateScriptOptions{
			Content: exampleValidContent,
		})

		assert.Equal(t, "postman/collection", name)
	})

}
