package testkube_api_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTestkubeApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TestkubeApi Suite")
}
