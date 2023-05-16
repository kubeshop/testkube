package smoke_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Smoke", func() {
	Describe("Ginkgo smoke test", func() {
		It("Positive test - should always pass", func() {
			Expect(true).To(Equal(true))
		})
	})
})
