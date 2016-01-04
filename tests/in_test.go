package email_resource_test

import (
	. "github.com/pivotal-cf/email-resource/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/pivotal-cf/email-resource/Godeps/_workspace/src/github.com/onsi/gomega"
)

var _ = Describe("In", func() {
	BeforeEach(func() {
		Run("go", "build", "-o", "../bin/in", "../actions/in")
	})

	Context("when the version is given on Standard In", func() {
		It("should output the version that it was given", func() {
			inputData := `
{
  "source": {
    "a-key": "some data",
    "another-key": "some more data"
  },
  "version": { "ref": "61cebf" }
}
			`
			output := RunWithStdin(inputData, "../bin/in")

			Expect(output).To(MatchJSON(`{"version": { "ref": "61cebf" }}`))
		})
	})

	Context("when the version is not given on stdin", func() {
		It("should print an error and exit 1", func() {
			output, err := RunWithStdinAllowError(`{ "missing" : "the version" }`, "../bin/in")
			Expect(err).To(MatchError("exit status 1"))
			Expect(output).To(ContainSubstring("missing version"))
		})
	})

})
