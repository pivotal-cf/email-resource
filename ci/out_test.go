package email_resource_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/email-resource/ci/fakes"
)

var _ = Describe("Out", func() {
	var inputdata string
	var smtpServer *fakes.SMTP
	type MetadataItem struct {
		Name  string
		Value string
	}
	var structuredInputData struct {
		Source struct {
			SMTP struct {
				Host     string `json:"host"`
				Port     int    `json:"port"`
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

	toCleanup := make(map[string]string)

	makeTempFile := func(label, contents string) string {
		fh, err := ioutil.TempFile("", label)
		Expect(err).NotTo(HaveOccurred())
		fh.WriteString(contents)
		fh.Close()
		toCleanup[label] = fh.Name()
		return fh.Name()
	}

	BeforeEach(func() {
		Run("go", "build", "-o", "../bin/out", "../actions/out")
		smtpServer = fakes.NewSMTP()
		smtpServer.Boot()

		var err error
		structuredInputData.Source.SMTP.Username = "some username"
		structuredInputData.Source.SMTP.Password = "some password"
		structuredInputData.Source.SMTP.Host = smtpServer.Host
		structuredInputData.Source.SMTP.Port, err = strconv.Atoi(smtpServer.Port)
		Expect(err).NotTo(HaveOccurred())

		structuredInputData.Source.To = []string{"recipient@example.com", "recipient+2@example.com"}
		structuredInputData.Source.From = "sender@example.com"

		structuredInputData.Params.Subject = makeTempFile("subject", "some subject line")
		structuredInputData.Params.Body = makeTempFile("body", `this is a body
it has many lines

even empty lines

!`)

		inputBytes, err := json.Marshal(structuredInputData)
		Expect(err).NotTo(HaveOccurred())
		inputdata = string(inputBytes)
	})

	AfterEach(func() {
		smtpServer.Close()
		for _, filename := range toCleanup {
			os.Remove(filename)
		}
	})

	It("should report the current time as a version and exit 0", func() {
		output, err := RunWithStdinAllowError(inputdata, "../bin/out", "some", "arguments")
		Expect(err).NotTo(HaveOccurred())

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
		output, err := RunWithStdinAllowError(inputdata, "../bin/out")
		Expect(err).NotTo(HaveOccurred())

		var outdata struct {
			Metadata []MetadataItem
		}
		Expect(json.Unmarshal([]byte(output), &outdata)).To(Succeed())
		Expect(outdata.Metadata).To(ContainElement(Equal(MetadataItem{Name: "smtp_host", Value: smtpServer.Host})))
		Expect(outdata.Metadata).To(ContainElement(Equal(MetadataItem{Name: "subject", Value: "some subject line"})))
	})

	It("should send an email", func() {
		_, err := RunWithStdinAllowError(inputdata, "../bin/out")
		Expect(err).NotTo(HaveOccurred())

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
			structuredInputData.Params.Body = ""
			inputBytes, err := json.Marshal(structuredInputData)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)

			_, err = RunWithStdinAllowError(inputdata, "../bin/out")
			Expect(err).NotTo(HaveOccurred())

			Expect(smtpServer.Deliveries).To(HaveLen(1))
			delivery := smtpServer.Deliveries[0]
			Expect(delivery.Data).To(HaveSuffix("Subject: some subject line\n"))
		})
	})

	Context("when the 'From' is empty", func() {
		It("should print an error and exit 1", func() {
			structuredInputData.Source.From = ""
			inputBytes, err := json.Marshal(structuredInputData)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)

			output, err := RunWithStdinAllowError(inputdata, "../bin/out")
			Expect(err).To(MatchError("exit status 1"))
			Expect(output).To(Equal(`missing required field "source.from"`))
		})
	})

	Context("when the 'To' is empty", func() {
		It("should print an error and exit 1", func() {
			structuredInputData.Source.To = nil
			inputBytes, err := json.Marshal(structuredInputData)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)

			output, err := RunWithStdinAllowError(inputdata, "../bin/out")
			Expect(err).To(MatchError("exit status 1"))
			Expect(output).To(Equal(`missing required field "source.to"`))
		})
	})

	Context("when the 'source.smtp.username' is empty", func() {
		It("should print an error and exit 1", func() {
			structuredInputData.Source.SMTP.Username = ""
			inputBytes, err := json.Marshal(structuredInputData)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)

			output, err := RunWithStdinAllowError(inputdata, "../bin/out")
			Expect(err).To(MatchError("exit status 1"))
			Expect(output).To(Equal(`missing required field "source.smtp.username"`))
		})
	})

	Context("when the 'source.smtp.password' is empty", func() {
		It("should print an error and exit 1", func() {
			structuredInputData.Source.SMTP.Password = ""
			inputBytes, err := json.Marshal(structuredInputData)
			Expect(err).NotTo(HaveOccurred())
			inputdata = string(inputBytes)

			output, err := RunWithStdinAllowError(inputdata, "../bin/out")
			Expect(err).To(MatchError("exit status 1"))
			Expect(output).To(Equal(`missing required field "source.smtp.password"`))
		})
	})

	Context("When the STDIN is not valid JSON", func() {
		It("should print an error and exit 1", func() {
			output, err := RunWithStdinAllowError("not JSON", "../bin/out", "some", "arguments")
			Expect(err).To(MatchError("exit status 1"))
			Expect(output).To(HavePrefix("error parsing input as JSON: "))
		})
	})

})
