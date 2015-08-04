package email_resource_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Out", func() {
	var inputdata string
	type MetadataItem struct {
		Name  string
		Value string
	}

	BeforeEach(func() {
		Run("go", "build", "-o", "../bin/out", "../actions/out")
		inputdata = `
{
  "source": {
    "smtp": {
			"host": "smtp.sendgrid.net"
		}
  },
  "params": {
		"subject": "/some/path/to/subject/line"
  }
}
		`
	})

	It("should report the current time as a version and exit 0", func() {
		output, err := RunWithStdinAllowError("{}", "../bin/out", "some", "arguments")
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
		Expect(outdata.Metadata).To(ContainElement(Equal(MetadataItem{Name: "smtp_host", Value: "smtp.sendgrid.net"})))
		Expect(outdata.Metadata).To(ContainElement(Equal(MetadataItem{Name: "subject", Value: "/some/path/to/subject/line"})))
	})

	Context("When the STDIN is not valid JSON", func() {
		It("should print an error and exit 1", func() {
			output, err := RunWithStdinAllowError("not JSON", "../bin/out", "some", "arguments")
			Expect(err).To(HaveOccurred())
			Expect(output).To(Equal("expected JSON input"))
		})
	})

})
