package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Smoke", func() {
	Describe("Ginkgo smoke test", func() {
		It("Negative test - should always fail", func(){
			Expect(true).To(Equal(false))
		})
	})
})
