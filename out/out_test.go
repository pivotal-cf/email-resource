package out_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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

	Describe("Using custom certificates", func() {
		var smtpServerCa *FakeSMTPServer

		BeforeEach(func() {
			smtpServerCa = NewFakeSMTPServerWithCustomCert("./test_certs/server.crt", "./test_certs/server.key")
			smtpServerCa.Boot()

			inputs.Source.SMTP.Host = smtpServerCa.Host
			inputs.Source.SMTP.Port = smtpServerCa.Port
		})

		AfterEach(func() {
			smtpServerCa.Close()
		})

		Context("when no custom certificate is configured", func() {
			failWithCertificateError := func() {
				output, err := out.Execute(sourceRoot, "the-version", []byte(inputdata))

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(BeEquivalentTo(`unable to start TLS: x509: certificate signed by unknown authority`))
				Expect(output).Should(BeEmpty())
			}

			BeforeEach(func() {
				inputs.Source.SMTP.CaCert = ""
			})

			It("fails with an error", failWithCertificateError)

			Context("when 'anonymous' is 'true'", func() {
				BeforeEach(func() {
					inputs.Source.SMTP.Anonymous = true
				})

				It("still fails with an error", failWithCertificateError)
			})
		})

		Context("when a custom certificate is configured", func() {
			BeforeEach(func() {
				caCert, err := ioutil.ReadFile("./test_certs/rootCA.pem")
				if err != nil {
					panic(err)
				}

				inputs.Source.SMTP.CaCert = string(caCert)
			})

			It("can connect to the server", func() {
				output, err := out.Execute(sourceRoot, "the-version", []byte(inputdata))

				Expect(err).ToNot(HaveOccurred())
				Expect(output).ToNot(BeEmpty())
			})
		})
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
		Expect(string(delivery.Data)).To(ContainSubstring("To: recipient@example.com,recipient+2@example.com,recipient+3@example.com"))
		Expect(string(delivery.Data)).To(ContainSubstring("some subject line"))
		Expect(string(delivery.Data)).To(ContainSubstring(`this is a body
it has many lines

even empty lines

!`))
	})

	Context("when the 'Subject' parameter is empty", func() {
		BeforeEach(func() {
			inputs.Params.Subject = ""
		})

		Context("but the 'SubjectText' paramter is given", func() {
			BeforeEach(func() {
				inputs.Params.SubjectText = "some subject line"
			})

			It("succeeds and sends an email", func() {
				output, err := out.Execute(sourceRoot, "the-version", []byte(inputdata))
				Expect(err).ToNot(HaveOccurred())
				Expect(output).ToNot(BeEmpty())

				Expect(smtpServer.Deliveries).To(HaveLen(1))
				delivery := smtpServer.Deliveries[0]
				Expect(string(delivery.Data)).To(ContainSubstring("some subject line"))
			})
		})

		Context("and the 'SubjectText' paramter is empty", func() {
			It("returns an error", func() {
				inputBytes, err := json.Marshal(inputs)
				Expect(err).NotTo(HaveOccurred())

				output, err := out.Execute(sourceRoot, "", inputBytes)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(BeEquivalentTo(`Invalid configuration: missing required field "params.subject" or "params.subject_text". Must specify at least one`))
				Expect(output).To(BeEmpty())
				Expect(smtpServer.Deliveries).To(HaveLen(0))
			})
		})
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
			Expect(string(delivery.Data)).To(ContainSubstring("some subject line"))

		})
	})

	Context("when the subject has template syntax", func() {
		BeforeEach(func() {
			os.Setenv("BUILD_ID", "5")
			createSource(inputs.Params.Subject, "some subject line for #${BUILD_ID}")
		})
		var subject string

		verifyTemplateInterpolation := func() {
			out.Execute(sourceRoot, "", []byte(inputdata))
			Expect(smtpServer.Deliveries).To(HaveLen(1))
			delivery := smtpServer.Deliveries[0]
			Expect(string(delivery.Data)).To(ContainSubstring("some subject line for #5"))
		}

		BeforeEach(func() {
			os.Setenv("BUILD_ID", "5")
			subject = "some subject line for #${BUILD_ID}"
		})

		Context("when the subject is given as file", func() {
			BeforeEach(func() {
				createSource(inputs.Params.Subject, subject)
			})

			It("interpolates the template", verifyTemplateInterpolation)
		})

		Context("when the subject is given as text", func() {
			BeforeEach(func() {
				inputs.Params.Subject = ""
				inputs.Params.SubjectText = subject
			})

			It("interpolates the template", verifyTemplateInterpolation)
		})
	})

	Context("when a headers file is provided", func() {
		var headers string

		BeforeEach(func() {
			headers = `Header-1: value
Header-2: value`

			headersFilePath := "some/path/to/headers.txt"
			createSource(headersFilePath, headers)
			inputs.Params.Headers = headersFilePath
		})

		It("should add the headers to the email", func() {
			out.Execute(sourceRoot, "", []byte(inputdata))

			Expect(smtpServer.Deliveries).To(HaveLen(1))
			delivery := smtpServer.Deliveries[0]
			Expect(string(delivery.Data)).To(ContainSubstring("Header-1: value\n"))
			Expect(string(delivery.Data)).To(ContainSubstring("Header-2: value\n"))
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
				Expect(string(delivery.Data)).To(ContainSubstring("Header-1: value-1\n"))
				Expect(string(delivery.Data)).To(ContainSubstring("Header-2: value-2\n"))
				Expect(string(delivery.Data)).To(ContainSubstring("Header-3: value-3\n"))
			})
		})
	})

	Context("body", func() {

		verifyBody := func(expectedBody string) func() {
			return func() {
				output, err := out.Execute(sourceRoot, "", []byte(inputdata))
				Expect(err).ToNot(HaveOccurred())
				Expect(output).ToNot(BeEmpty())

				Expect(smtpServer.Deliveries).To(HaveLen(1))
				delivery := smtpServer.Deliveries[0]

				Expect(string(delivery.Data)).To(ContainSubstring(expectedBody))
			}
		}

		Context("when body file is provided", func() {
			BeforeEach(func() {
				createSource(inputs.Params.Body, "some body")
				inputs.Params.BodyText = ""
			})

			It("uses the text from the body file", verifyBody("some body"))
		})

		Context("when body text is provided", func() {
			BeforeEach(func() {
				inputs.Params.Body = ""
				inputs.Params.BodyText = "some body"
			})
			It("uses the body text", verifyBody("some body"))
		})

		Context("when body text and body file is provided", func() {
			BeforeEach(func() {
				createSource(inputs.Params.Body, "some body from file")
				inputs.Params.BodyText = "some body from text"
			})
			It("uses the body text", verifyBody("some body from text"))
		})

	})

	Context("when the body and the body_text is empty", func() {
		BeforeEach(func() {
			inputs.Params.Body = ""
			inputs.Params.BodyText = ""
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
				Expect(string(delivery.Data)).To(ContainSubstring("some subject line"))
			})
		})
		Context("when the 'SendEmptyBody' parameter is false", func() {
			BeforeEach(func() {
				inputs.Params.SendEmptyBody = false
			})

			It("should not return an error and not send a message", func() {
				inputBytes, err := json.Marshal(inputs)
				Expect(err).NotTo(HaveOccurred())

				output, err := out.Execute(sourceRoot, "", inputBytes)
				Expect(err).ShouldNot(HaveOccurred())
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
			Expect(err.Error()).To(BeEquivalentTo(`Invalid configuration: missing required field "source.from"`))
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
				Expect(err.Error()).To(BeEquivalentTo(`Invalid configuration: missing required field "source.to" or "params.to" or "params.to_text". Must specify at least one`))
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
			Expect(err.Error()).To(BeEquivalentTo(`Invalid configuration: missing required field "source.smtp.username" if anonymous specify anonymous: true`))
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
			Expect(err.Error()).To(BeEquivalentTo(`Invalid configuration: missing required field "source.smtp.password" if anonymous specify anonymous: true`))
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
			Expect(err.Error()).To(BeEquivalentTo("unmarshalling input: unexpected end of JSON input"))
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
			Expect(err.Error()).To(ContainSubstring("connection refused"))
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
