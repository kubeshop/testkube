package executortests

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

Describe("Ginkgo smoke test", func() {
    It("Should always pass", func(){
		Expect(true).To(Equal(true))
    })
})