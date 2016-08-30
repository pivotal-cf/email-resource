package email_resource_test

import (
	"os"

	"github.com/pivotal-cf/email-resource/actions/out"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metadata Template", func() {

	Describe(".RenderTemplate", func(){
		concourseEnvVars := [...]string{"BUILD_ID", "BUILD_NAME", "BUILD_JOB_NAME", "BUILD_PIPELINE_NAME", "ATC_EXTERNAL_URL"}

		for _, envVar := range concourseEnvVars {

			It("replaces ." + envVar, func (envVar string) func() {
				return func() {
					value := envVar
					os.Setenv(envVar, value)
					text := "${" + envVar + "}"
					GinkgoWriter.Write([]byte(text))
					GinkgoWriter.Write([]byte(value))

					renderedText := out.RenderTemplate(text)
					Expect(renderedText).To(Equal(value))
				}
			}(envVar))

		}

		It("replaces .BuildJobName with other value", func(){
			os.Setenv("BUILD_JOB_NAME", "other build job name value")
			text := "${BUILD_JOB_NAME}"

			renderedText := out.RenderTemplate(text)
			Expect(renderedText).To(Equal("other build job name value"))
		})

		Context("when the template is empty", func(){
			It("returns empty string", func(){
				text := ""

				renderedText := out.RenderTemplate(text)
				Expect(renderedText).To(Equal(""))
			})
		})

		Context("when a template cannot be parsed", func(){
			It("returns the original body", func(){
				text := "This ${BUILD_JOB_NAME is invalid"

				renderedText := out.RenderTemplate(text)
				Expect(renderedText).To(Equal("This ${BUILD_JOB_NAME is invalid"))
			})
		})

		Context("when a template contains an unknown template function", func(){
			It("returns the original text", func(){
				text := "This ${IS_UNKNOWN} is unknown"

				renderedText := out.RenderTemplate(text)
				Expect(renderedText).To(Equal("This ${IS_UNKNOWN} is unknown"))
			})
		})
	})
})
