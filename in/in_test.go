package in_test

import (
	"crypto/tls"
	"fmt"
	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func contentsAt(path string) string {
	contents, err := ioutil.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())
	return strings.TrimSpace(string(contents))
}

func startServer() (*server.Server, error) {
	be := memory.New()
	user, err := be.Login("username", "password")
	Expect(err).NotTo(HaveOccurred())

	inbox, err := user.GetMailbox("INBOX")
	Expect(err).NotTo(HaveOccurred())

	header := `Content-Type: multipart/mixed; boundary=message-boundary
Date: Wed, 11 May 2016 14:31:59 +0000
From: contact@example.org
Message-Id: 42@example.org
Subject: A little message, just for you
To: contact@example.org
`
	textBody := `
--message-boundary
Content-Disposition: inline
Content-Type: text/plain

Hi there :)

--message-boundary
Content-Disposition: attachment; filename=attachment.txt
Content-Type: text/plain

some-attachment-data

--message-boundary--

`

	inbox.CreateMessage([]string{}, time.Now(), strings.NewReader(header+textBody))

	s := server.New(be)
	s.Addr = "127.0.0.1:9875"

	certs, err := tls.LoadX509KeyPair("./fixtures/server.crt", "./fixtures/server.key")
	if err != nil {
		return nil, err
	}

	s.TLSConfig = &tls.Config{Certificates: []tls.Certificate{certs}}

	fmt.Fprintln(GinkgoWriter, "Starting IMAP server at localhost:9875")
	go func() {
		if err := s.ListenAndServeTLS(); err != nil {
			fmt.Fprintf(GinkgoWriter, "[ERROR]: %s", err)
		}
	}()

	return s, nil
}

var _ = Describe("In", func() {
	var (
		destinationDir string
		server         *server.Server
		input          = `{
	          "source": {
	             "imap" : {
	             "host": "localhost",
                 "port": "9875",
                 "username": "username",
                 "password": "password",
	             "inbox": "INBOX",
                 "skip_ssl_validation": true
	            }
	          },
          	  "version": { "uid": "7" }
	        }`
	)

	BeforeEach(func() {
		var err error
		server, err = startServer()
		Expect(err).NotTo(HaveOccurred())

		destinationDir, err = ioutil.TempDir("", "email_resource_test_")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
		os.RemoveAll(destinationDir)
	})

	It("works", func() {
		command := exec.Command(binaryPath, destinationDir)
		command.Stdin = strings.NewReader(input)

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "10s").Should(gexec.Exit(0))

		output := session.Out.Contents()

		//memory backend implementation start at uid 6
		Expect(string(output)).Should(MatchJSON(`
        	{
        	  "version" : {"uid": "7"},
        	  "metadata": [
                    {
                      "name": "Subject",
                      "value": "A little message, just for you"
                    },
                    {
                      "name": "Date",
                      "value": "Wednesday, 11-May-16 14:31:59 +0000"
                    },
                    {
                      "name": "Number of Attachments",
                      "value": "1"
                    }
                  ]
        	}`))
	})

	It("writes the correct files to the destination directory", func() {
		command := exec.Command(binaryPath, destinationDir)
		command.Stdin = strings.NewReader(input)

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "10s").Should(gexec.Exit(0))

		subjectPath := filepath.Join(destinationDir, "subject")
		Expect(contentsAt(subjectPath)).To(Equal("A little message, just for you"))
		versionPath := filepath.Join(destinationDir, "version")
		Expect(contentsAt(versionPath)).To(Equal("42@example.org"))
		datePath := filepath.Join(destinationDir, "date")
		Expect(contentsAt(datePath)).To(Equal("Wednesday, 11-May-16 14:31:59 +0000"))
		bodyPath := filepath.Join(destinationDir, "body")
		Expect(contentsAt(bodyPath)).To(Equal("Hi there :)"))

		attachmentPath := filepath.Join(destinationDir, "attachments", "attachment.txt")
		Expect(contentsAt(attachmentPath)).To(Equal("some-attachment-data"))
	})
})
