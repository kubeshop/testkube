package executortests

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

Describe("Ginkgo smoke test", func() {
    It("Negative test - should always fail", func(){
		Expect(true).To(Equal(false))
    })
})