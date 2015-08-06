package email_resource_test

import (
	"encoding/json"
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
	var inputs struct {
		Source struct {
			SMTP struct {
				Host     string `json:"host"`
				Port     string `json:"port"`
				Username string `json:"username"`
				Password string `json:"password"`
			} `json:"smtp"`
			To   []string `json:"to"`
			From string   `json:"from"`
		} `json:"source"`
		Params struct {
			Subject string `json:"subject"`
			Body    string `json:"body"`
		} `json:"params"`
	}

	createSource := func(relativePath, contents string) {
		absPath := path.Join(sourceRoot, relativePath)
		Expect(os.MkdirAll(filepath.Dir(absPath), 0700)).To(Succeed())
		Expect(ioutil.WriteFile(absPath, []byte(contents), 0600)).To(Succeed())
	}

	BeforeEach(func() {
		Run("go", "build", "-o", "../bin/out", "../actions/out")
		smtpServer = NewFakeSMTPServer()
		smtpServer.Boot()

		var err error
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
		Expect(data).To(ContainElement("Subject: some subject line"))
		Expect(string(delivery.Data)).To(ContainSubstring(`this is a body
it has many lines

even empty lines

!`))
	})

	Context("when the body is empty", func() {
		It("should succeed and send a message with an empty body", func() {
			inputs.Params.Body = ""
			inputBytes, err := json.Marshal(inputs)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)

			RunWithStdin(inputdata, "../bin/out", sourceRoot)

			Expect(smtpServer.Deliveries).To(HaveLen(1))
			delivery := smtpServer.Deliveries[0]
			Expect(delivery.Data).To(HaveSuffix("Subject: some subject line\n"))
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
			Expect(output).To(Equal(`missing required field "source.smtp.username"`))
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
			Expect(output).To(Equal(`missing required field "source.smtp.password"`))
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
