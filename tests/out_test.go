package email_resource_test

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
)

var _ = Describe("Out", func() {
	var inputdata string
	var sourceRoot string
	var smtpServer *FakeSMTPServer
	type MetadataItem struct {
		Name  string
		Value string
	}

	type inputStruct struct {
		Source struct {
			SMTP struct {
				Host      string `json:"host"`
				Port      string `json:"port"`
				Username  string `json:"username"`
				Password  string `json:"password"`
				Anonymous bool   `json:"anonymous"`
			} `json:"smtp"`
			To   []string `json:"to"`
			From string   `json:"from"`
		} `json:"source"`
		Params struct {
			Subject       string `json:"subject"`
			Body          string `json:"body"`
			SendEmptyBody bool   `json:"send_empty_body"`
			Headers       string `json:"headers"`
		} `json:"params"`
	}

	var inputs inputStruct

	createSource := func(relativePath, contents string) {
		absPath := path.Join(sourceRoot, relativePath)
		Expect(os.MkdirAll(filepath.Dir(absPath), 0700)).To(Succeed())
		Expect(ioutil.WriteFile(absPath, []byte(contents), 0600)).To(Succeed())
	}

	BeforeSuite(func() {
		Run("go", "build", "-o", "../bin/out", "../actions/out")
	})

	BeforeEach(func() {
		smtpServer = NewFakeSMTPServer()
		smtpServer.Boot()

		var err error
		inputs = inputStruct{}
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
		createSource(inputs.Params.Subject, "some subject line")
		createSource(inputs.Params.Body, `this is a body
it has many lines

even empty lines

!`)
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
		output := RunWithStdin(inputdata, "../bin/out", sourceRoot)

		var outdata struct {
			Version struct {
				Time time.Time
			}
		}
		Expect(json.Unmarshal([]byte(output), &outdata)).To(Succeed())
		Expect(outdata.Version.Time).To(BeTemporally("~", time.Now(), 5*time.Second))

		var untyped map[string]interface{}
		Expect(json.Unmarshal([]byte(output), &untyped)).To(Succeed())
		Expect(untyped).To(HaveKey("version"))
	})

	It("should report all the expected metadata fields", func() {
		output := RunWithStdin(inputdata, "../bin/out", sourceRoot)

		var outdata struct {
			Metadata []MetadataItem
		}
		Expect(json.Unmarshal([]byte(output), &outdata)).To(Succeed())
		Expect(outdata.Metadata).To(ContainElement(Equal(MetadataItem{Name: "smtp_host", Value: smtpServer.Host})))
		Expect(outdata.Metadata).To(ContainElement(Equal(MetadataItem{Name: "subject", Value: "some subject line"})))
	})

	It("should send an email", func() {
		RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)

		Expect(smtpServer.Deliveries).To(HaveLen(1))
		delivery := smtpServer.Deliveries[0]
		Expect(delivery.Sender).To(Equal("sender@example.com"))
		Expect(delivery.Recipients).To(Equal([]string{"recipient@example.com", "recipient+2@example.com"}))

		data := strings.Split(string(delivery.Data), "\n")
		Expect(data).To(ContainElement("To: recipient@example.com, recipient+2@example.com"))
		Expect(data).To(ContainElement("Subject: some subject line"))
		Expect(string(delivery.Data)).To(ContainSubstring(`this is a body
it has many lines

even empty lines

!`))
	})

	It("makes sure that all headers are separate by one newline", func() {
		RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)
		Expect(smtpServer.Deliveries).To(HaveLen(1))
		delivery := smtpServer.Deliveries[0]
		Expect(delivery.Data).To(BeEquivalentTo(`To: recipient@example.com, recipient+2@example.com
From: sender@example.com
Subject: some subject line

this is a body
it has many lines

even empty lines

!
`))

	})

	It("adds a extra newline between the last header and the body", func() {
		RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)
		Expect(smtpServer.Deliveries).To(HaveLen(1))
		delivery := smtpServer.Deliveries[0]
		Expect(delivery.Data).To(ContainSubstring(`Subject: some subject line

this is a body
it has many lines

even empty lines

!`))

	})

	Context("when the subject has an extra newline", func() {
		BeforeEach(func() {
			createSource(inputs.Params.Subject, "some subject line\n\n")
		})

		It("strips the extra newline", func() {
			RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)
			Expect(smtpServer.Deliveries).To(HaveLen(1))
			delivery := smtpServer.Deliveries[0]
			Expect(delivery.Data).To(ContainSubstring(`Subject: some subject line

this is a body
it has many lines

even empty lines

!`))

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
			RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)

			Expect(smtpServer.Deliveries).To(HaveLen(1))
			delivery := smtpServer.Deliveries[0]

			data := strings.Split(string(delivery.Data), "\n")
			Expect(data).To(ContainElement("Header-1: value"))
			Expect(data).To(ContainElement("Header-2: value"))
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
				RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)
				Expect(smtpServer.Deliveries).To(HaveLen(1))
				delivery := smtpServer.Deliveries[0]
				Expect(delivery.Data).To(ContainSubstring(`Header-1: value-1
Header-2: value-2
Header-3: value-3
Subject: some subject line

`))
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
				inputdata = string(inputBytes)

				RunWithStdin(inputdata, "../bin/out", sourceRoot)

				Expect(smtpServer.Deliveries).To(HaveLen(1))
				delivery := smtpServer.Deliveries[0]
				Expect(delivery.Data).To(HaveSuffix("Subject: some subject line\n\n"))
			})
		})
		Context("when the 'SendEmptyBody' parameter is false", func() {
			BeforeEach(func() {
				inputs.Params.SendEmptyBody = false
			})
			It("should succeed and not send a message", func() {
				inputBytes, err := json.Marshal(inputs)
				Expect(err).NotTo(HaveOccurred())
				inputdata = string(inputBytes)

				RunWithStdin(inputdata, "../bin/out", sourceRoot)

				Expect(smtpServer.Deliveries).To(HaveLen(0))
			})

			It("should print a message to stderr", func() {
				inputBytes, err := json.Marshal(inputs)
				Expect(err).NotTo(HaveOccurred())
				inputdata = string(inputBytes)

				output := RunWithStdin(inputdata, "../bin/out", sourceRoot)
				Expect(output).To(ContainSubstring("Message not sent because the message body is empty and send_empty_body parameter was set to false. Github readme: https://github.com/pivotal-cf/email-resource"))
			})
		})
	})

	Context("when the 'From' is empty", func() {
		It("should print an error and exit 1", func() {
			inputs.Source.From = ""
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)

			output, err := RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)
			Expect(err).To(MatchError("exit status 1"))
			Expect(output).To(Equal(`missing required field "source.from"`))
		})
	})

	Context("when the 'To' is empty", func() {
		It("should print an error and exit 1", func() {
			inputs.Source.To = nil
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)

			output, err := RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)
			Expect(err).To(MatchError("exit status 1"))
			Expect(output).To(Equal(`missing required field "source.to"`))
		})
	})

	Context("when the 'source.smtp.username' is empty", func() {
		It("should print an error and exit 1", func() {
			inputs.Source.SMTP.Username = ""
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)

			output, err := RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)
			Expect(err).To(MatchError("exit status 1"))
			Expect(output).To(Equal(`missing required field "source.smtp.username" if anonymous specify anonymous: true`))
		})
	})

	Context("when the 'source.smtp.username' is empty and Anonymous", func() {
		It("should not error", func() {
			inputs.Source.SMTP.Username = ""
			inputs.Source.SMTP.Anonymous = true
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)

			_, err = RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("when the 'source.smtp.password' is empty", func() {
		It("should print an error and exit 1", func() {
			inputs.Source.SMTP.Password = ""
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)

			output, err := RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)
			Expect(err).To(MatchError("exit status 1"))
			Expect(output).To(Equal(`missing required field "source.smtp.password" if anonymous specify anonymous: true`))
		})
	})

	Context("when the 'source.smtp.password' is empty and Anonymous", func() {
		It("should not error", func() {
			inputs.Source.SMTP.Password = ""
			inputs.Source.SMTP.Anonymous = true
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)

			_, err = RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("When the STDIN is not valid JSON", func() {
		It("should print an error and exit 1", func() {
			output, err := RunWithStdinAllowError("not JSON", "../bin/out", sourceRoot)
			Expect(err).To(MatchError("exit status 1"))
			Expect(output).To(HavePrefix("error parsing input as JSON: "))
		})
	})

	Context("when a sourceRoot is not provided as the first command-line argument", func() {
		It("should print an error and exit 1", func() {
			output, err := RunWithStdinAllowError(inputdata, "../bin/out", "")
			Expect(output).To(Equal("expected path to build sources as first argument"))
			Expect(err).To(MatchError("exit status 1"))
		})
	})

	Context("when smtp server is not available", func() {
		It("should print an error and exit 1", func() {
			inputs.Source.SMTP.Port = "1111"
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)
			output, err := RunWithStdinAllowError(inputdata, "../bin/out", sourceRoot)
			Expect(err).To(MatchError("exit status 1"))
			Expect(output).To(Equal(fmt.Sprintf("Unable to send an email using SMTP server %s"+
				" on port %s: dial tcp %s:%s: getsockopt: connection refused", inputs.Source.SMTP.Host, inputs.Source.SMTP.Port, inputs.Source.SMTP.Host, inputs.Source.SMTP.Port)))
		})
	})

	Context("when the subject file is an absolute path", func() {
		It("should succeed", func() {
			var err error
			inputs.Params.Subject, err = filepath.Abs(filepath.Join(sourceRoot, "some/path/to/subject.txt"))
			Expect(err).NotTo(HaveOccurred())
			inputBytes, _ := json.Marshal(inputs)

			RunWithStdin(string(inputBytes), "../bin/out", sourceRoot)
		})
	})
})
