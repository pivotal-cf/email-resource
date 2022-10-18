package out_test

import (
	"bytes"

	"github.com/domodwyer/mailyak/v3"
	"github.com/pivotal-cf/email-resource/out"
	"github.com/pivotal-cf/email-resource/out/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sender", func() {
	Context("Adding Headers", func() {
		var mailfake *fakes.FakeMail
		var mailCreator out.MailCreator
		BeforeEach(func() {
			mailfake = &fakes.FakeMail{}
			mailfake.PlainReturns(&mailyak.BodyPart{})
			mailfake.HTMLReturns(&mailyak.BodyPart{})
			mailfake.MimeBufReturns(&bytes.Buffer{}, nil)
			mailCreator.Mail = mailfake
		})
		It("Will not add mime and content type headers", func() {
			mailCreator.AddHeader("MIME-version", "1.0")
			mailCreator.AddHeader("Content-Type", "text/html; charset=\"UTF-8\"")
			_, err := mailCreator.Compose()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(mailfake.AddHeaderCallCount()).Should(Equal(0))
		})

		It("Will use HTML body part if header is found", func() {
			mailCreator.AddHeader("Content-Type", "text/html; charset=\"UTF-8\"")
			_, err := mailCreator.Compose()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(mailfake.AddHeaderCallCount()).Should(Equal(0))
			Expect(mailfake.PlainCallCount()).Should(Equal(0))
			Expect(mailfake.HTMLCallCount()).Should(Equal(1))
		})
	})
})
