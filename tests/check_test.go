package email_resource_test

import (
	. "github.com/pivotal-cf/email-resource/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/pivotal-cf/email-resource/Godeps/_workspace/src/github.com/onsi/gomega"
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
