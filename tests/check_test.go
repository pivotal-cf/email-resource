package email_resource_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Check", func() {
	BeforeEach(func() {
		Run("go", "build", "-o", "../bin/check", "../actions/check")
	})

	It("should output an empty JSON list", func() {
		output := Run("../bin/check")
		Expect(output).To(MatchJSON("[]"))
	})
})
