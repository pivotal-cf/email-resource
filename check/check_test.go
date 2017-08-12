package check_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/email-resource/check"
)

var _ = Describe("Check", func() {
	AfterEach(func() {
		CleanupBuildArtifacts()
	})

	It("should compile", func() {
		_, err := Build("github.com/pivotal-cf/email-resource/check/cmd")
		Ω(err).ShouldNot(HaveOccurred())
	})

	It("should output an empty JSON list", func() {
		output, err := check.Execute()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).Should(MatchJSON("[]"))
	})
})
