package detector

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

func TestDetector(t *testing.T) {

	t.Run("detect postman/collection", func(t *testing.T) {

		detector := Detector{}
		detector.Add(CurlTestAdapter{})
		detector.Add(PostmanCollectionAdapter{})

		name, found := detector.Detect(client.UpsertScriptOptions{
			Content: testkube.NewStringScriptContent(exampleValidContent),
		})

		assert.True(t, found, "detector should find postman/collection")
		assert.Equal(t, "postman/collection", name)
	})

}
