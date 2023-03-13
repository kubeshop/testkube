package smoke_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"test"
)

var _ = Describe("Smoke", func() {
Describe("Ginkgo smoke test", func() {
    It("Positive test - should always pass", func(){
		Expect(true).To(Equal(pass))
    })
})
})
