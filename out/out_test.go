package out_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/email-resource/out"
)

var _ = Describe("Out", func() {
	var inputdata string
	var sourceRoot string
	var smtpServer *FakeSMTPServer

	var inputs out.Input

	createSource := func(relativePath, contents string) {
		absPath := path.Join(sourceRoot, relativePath)
		Expect(os.MkdirAll(filepath.Dir(absPath), 0700)).To(Succeed())
		Expect(ioutil.WriteFile(absPath, []byte(contents), 0600)).To(Succeed())
	}
	It("should compile", func() {
		_, err := Build("github.com/pivotal-cf/email-resource/out/cmd")
		Ω(err).ShouldNot(HaveOccurred())
	})
	BeforeEach(func() {
		smtpServer = NewFakeSMTPServer()
		smtpServer.Boot()

		var err error
		inputs = out.Input{}
		inputs.Source.SMTP.Username = "some username"
		inputs.Source.SMTP.Password = "some password"
		inputs.Source.SMTP.Host = smtpServer.Host
		inputs.Source.SMTP.Port = smtpServer.Port

		inputs.Source.To = []string{"recipient@example.com", "recipient+2@example.com"}
		inputs.Source.From = "sender@example.com"

		sourceRoot, err = ioutil.TempDir("", "sources")
		Expect(err).NotTo(HaveOccurred())

		inputs.Params.Subject = "some/path/to/subject.txt"
		inputs.Params.Body = "some/other/path/to/body"
		inputs.Params.To = "some/other/path/to/to"
		createSource(inputs.Params.Subject, "some subject line")
		createSource(inputs.Params.Body, `this is a body
it has many lines

even empty lines

!`)
		createSource(inputs.Params.To, "recipient+3@example.com")
	})

	JustBeforeEach(func() {
		inputBytes, err := json.Marshal(inputs)
		Expect(err).NotTo(HaveOccurred())
		inputdata = string(inputBytes)
	})

	AfterEach(func() {
		smtpServer.Close()
		os.RemoveAll(sourceRoot)
	})

	It("should report the current time as a version and exit 0", func() {
		output, err := out.Execute(sourceRoot, "the-version", []byte(inputdata))
		Expect(err).ToNot(HaveOccurred())
		Expect(output).ToNot(BeEmpty())
		var outdata out.Output
		Expect(json.Unmarshal([]byte(output), &outdata)).To(Succeed())
		Expect(outdata.Version.Time).To(BeTemporally("~", time.Now(), 5*time.Second))

		var untyped map[string]interface{}
		Expect(json.Unmarshal([]byte(output), &untyped)).To(Succeed())
		Expect(untyped).To(HaveKey("version"))
	})

	It("should report all the expected metadata fields", func() {
		output, err := out.Execute(sourceRoot, "the-version", []byte(inputdata))
		Expect(err).ToNot(HaveOccurred())
		Expect(output).ToNot(BeEmpty())
		var outdata out.Output
		Expect(json.Unmarshal([]byte(output), &outdata)).To(Succeed())
		Expect(outdata.Metadata).To(ContainElement(Equal(out.MetadataItem{Name: "smtp_host", Value: smtpServer.Host})))
		Expect(outdata.Metadata).To(ContainElement(Equal(out.MetadataItem{Name: "subject", Value: "some subject line"})))
		Expect(outdata.Metadata).To(ContainElement(Equal(out.MetadataItem{Name: "version", Value: "the-version"})))
	})

	It("should send an email", func() {
		output, err := out.Execute(sourceRoot, "the-version", []byte(inputdata))
		Expect(err).ToNot(HaveOccurred())
		Expect(output).ToNot(BeEmpty())

		Expect(smtpServer.Deliveries).To(HaveLen(1))
		delivery := smtpServer.Deliveries[0]
		Expect(delivery.Sender).To(Equal("sender@example.com"))
		Expect(delivery.Recipients).To(Equal([]string{"recipient@example.com", "recipient+2@example.com", "recipient+3@example.com"}))

		data := strings.Split(string(delivery.Data), "\n")
		Expect(data).To(ContainElement("To: recipient@example.com, recipient+2@example.com, recipient+3@example.com"))
		Expect(data).To(ContainElement("Subject: some subject line"))
		Expect(string(delivery.Data)).To(ContainSubstring(`this is a body
it has many lines

even empty lines

!`))
	})

	Context("when the subject has an extra newline", func() {
		BeforeEach(func() {
			createSource(inputs.Params.Subject, "some subject line\n\n")
		})

		It("strips the extra newline", func() {
			output, err := out.Execute(sourceRoot, "the-version", []byte(inputdata))
			Expect(err).ToNot(HaveOccurred())
			Expect(output).ToNot(BeEmpty())
			Expect(smtpServer.Deliveries).To(HaveLen(1))
			delivery := smtpServer.Deliveries[0]
			Expect(delivery.Data).To(ContainSubstring("Subject: some subject line"))

		})
	})

	Context("when the subject has template syntax", func() {
		BeforeEach(func() {
			os.Setenv("BUILD_ID", "5")
			createSource(inputs.Params.Subject, "some subject line for #${BUILD_ID}")
		})
	})

	Context("when a headers file is provided", func() {
		var headers string

		BeforeEach(func() {
			headers = `Header-1: value
Header-2: value
MIME-version: 4.0
Content-Type: text/html; charset="UTF-8"`

			headersFilePath := "some/path/to/headers.txt"
			createSource(headersFilePath, headers)
			inputs.Params.Headers = headersFilePath
		})

		It("should add the headers to the email", func() {
			out.Execute(sourceRoot, "", []byte(inputdata))

			Expect(smtpServer.Deliveries).To(HaveLen(1))
			delivery := smtpServer.Deliveries[0]
			Expect(delivery.Data).To(ContainSubstring("Header-1: value\n"))
			Expect(delivery.Data).To(ContainSubstring("Header-2: value\n"))
			Expect(delivery.Data).To(ContainSubstring("Mime-Version: 1.0\n"))
			Expect(delivery.Data).To(ContainSubstring("Content-Type: text/html; charset=UTF-8\n"))
			Expect(string(delivery.Data)).To(ContainSubstring(`
this is a body
it has many lines

even empty lines

!`))
		})

		Context("when a header has an extra newline", func() {
			BeforeEach(func() {
				headers = `Header-1: value-1
Header-2: value-2
Header-3: value-3


`
				createSource(inputs.Params.Headers, headers)
			})

			It("strips the extra newline", func() {
				out.Execute(sourceRoot, "", []byte(inputdata))
				Expect(smtpServer.Deliveries).To(HaveLen(1))
				delivery := smtpServer.Deliveries[0]
				Expect(delivery.Data).To(ContainSubstring("Header-1: value-1\n"))
				Expect(delivery.Data).To(ContainSubstring("Header-2: value-2\n"))
				Expect(delivery.Data).To(ContainSubstring("Header-3: value-3\n"))
			})
		})
	})

	Context("when the body is empty", func() {
		BeforeEach(func() {
			inputs.Params.Body = ""
		})

		Context("when the 'SendEmptyBody' parameter is true", func() {
			BeforeEach(func() {
				inputs.Params.SendEmptyBody = true
			})
			It("should succeed and send a message with an empty body", func() {
				inputBytes, err := json.Marshal(inputs)
				Expect(err).NotTo(HaveOccurred())

				output, err := out.Execute(sourceRoot, "", inputBytes)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(output).ShouldNot(BeEmpty())
				Expect(smtpServer.Deliveries).To(HaveLen(1))
				delivery := smtpServer.Deliveries[0]
				Expect(delivery.Data).To(ContainSubstring("Subject: some subject line\n"))
			})
		})
		Context("when the 'SendEmptyBody' parameter is false", func() {
			BeforeEach(func() {
				inputs.Params.SendEmptyBody = false
			})

			It("should return an error and not send a message", func() {
				inputBytes, err := json.Marshal(inputs)
				Expect(err).NotTo(HaveOccurred())

				output, err := out.Execute(sourceRoot, "", inputBytes)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(BeEquivalentTo("Message not sent because the message body is empty and send_empty_body parameter was set to false. Github readme: https://github.com/pivotal-cf/email-resource"))
				Expect(output).ShouldNot(BeEmpty())
				Expect(smtpServer.Deliveries).To(HaveLen(0))
			})
		})
	})

	Context("when the 'From' is empty", func() {
		It("should print an error and exit 1", func() {
			inputs.Source.From = ""
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())

			output, err := out.Execute(sourceRoot, "", inputBytes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(BeEquivalentTo(`missing required field "source.from"`))
			Expect(output).Should(BeEmpty())
		})
	})

	Context("when the 'To' is empty", func() {
		Context("When the to field is empty", func() {
			It("should print an error and exit 1", func() {
				inputs.Source.To = nil
				inputs.Params.To = ""
				inputBytes, err := json.Marshal(inputs)
				Expect(err).NotTo(HaveOccurred())

				output, err := out.Execute(sourceRoot, "", inputBytes)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(BeEquivalentTo(`missing required field "source.to" or "params.to". Must specify at least one`))
				Expect(output).Should(BeEmpty())
			})
		})

		Context("When the to field is not empty", func() {
			It("should succed", func() {
				inputs.Source.To = nil
				inputBytes, err := json.Marshal(inputs)
				Expect(err).NotTo(HaveOccurred())

				output, err := out.Execute(sourceRoot, "", inputBytes)
				Expect(err).ToNot(HaveOccurred())
				Expect(output).ShouldNot(BeEmpty())

			})
		})

	})

	Context("when the 'source.smtp.username' is empty", func() {
		It("should print an error and exit 1", func() {
			inputs.Source.SMTP.Username = ""
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())

			output, err := out.Execute(sourceRoot, "", inputBytes)
			Expect(err.Error()).To(BeEquivalentTo(`missing required field "source.smtp.username" if anonymous specify anonymous: true`))
			Expect(output).Should(BeEmpty())
		})
	})

	Context("when the 'source.smtp.username' is empty and Anonymous", func() {
		It("should not error", func() {
			inputs.Source.SMTP.Username = ""
			inputs.Source.SMTP.Anonymous = true
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())

			output, err := out.Execute(sourceRoot, "", inputBytes)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).ShouldNot(BeEmpty())
		})
	})

	Context("when the 'source.smtp.password' is empty", func() {
		It("should print an error and exit 1", func() {
			inputs.Source.SMTP.Password = ""
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())

			output, err := out.Execute(sourceRoot, "", inputBytes)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(BeEquivalentTo(`missing required field "source.smtp.password" if anonymous specify anonymous: true`))
			Expect(output).Should(BeEmpty())
		})
	})

	Context("when the 'source.smtp.password' is empty and Anonymous", func() {
		It("should not error", func() {
			inputs.Source.SMTP.Password = ""
			inputs.Source.SMTP.Anonymous = true
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())

			output, err := out.Execute(sourceRoot, "", inputBytes)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).ShouldNot(BeEmpty())
		})
	})

	Context("When the STDIN is not valid JSON", func() {
		It("should print an error and exit 1", func() {
			output, err := out.Execute(sourceRoot, "", []byte(""))
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(BeEquivalentTo("unexpected end of JSON input"))
			Expect(output).Should(BeEmpty())
		})
	})

	Context("when a sourceRoot is not provided as the first command-line argument", func() {
		It("should print an error and exit 1", func() {
			output, err := out.Execute("", "", []byte(inputdata))
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(BeEquivalentTo("expected path to build sources as first argument"))
			Expect(output).Should(BeEmpty())
		})
	})

	Context("when smtp server is not available", func() {
		It("should return an error", func() {
			inputs.Source.SMTP.Port = "1111"
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)
			output, err := out.Execute(sourceRoot, "", []byte(inputdata))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("dial tcp %s:%s: getsockopt: connection refused", inputs.Source.SMTP.Host, inputs.Source.SMTP.Port)))
			Expect(output).Should(BeEmpty())
		})
	})

	Context("when the subject file is an absolute path", func() {
		It("should succeed", func() {
			var err error
			inputs.Params.Subject, err = filepath.Abs(filepath.Join(sourceRoot, "some/path/to/subject.txt"))
			Expect(err).NotTo(HaveOccurred())
			inputBytes, _ := json.Marshal(inputs)

			output, err := out.Execute(sourceRoot, "", inputBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).ShouldNot(BeEmpty())
		})
	})
})
