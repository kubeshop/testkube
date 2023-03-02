package e2e

import (
	"flag"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var baseURL string

// Register your flags in an init function.  This ensures they are registered _before_ `go test` calls flag.Parse().
func init() {
	flag.StringVar(&baseURL, "base-url", "", "Url of the server to test")
}
func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	if baseURL == "" {
		baseURL = "google.com"
	}
	RunSpecs(t, "E2E Integration Testing Suite")
}
