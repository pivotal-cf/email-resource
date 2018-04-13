package check_test

import (
	"crypto/tls"
	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"os/exec"
	"strings"
	"time"
	"fmt"
)

func startServer() (*server.Server, error) {
	be := memory.New()
	user, err := be.Login("username", "password")
	Expect(err).NotTo(HaveOccurred())

	inbox, err := user.GetMailbox("INBOX")
	Expect(err).NotTo(HaveOccurred())

	body := `From: contact@example.org
To: contact@example.org
Subject: A little message, just for you
Date: Wed, 11 May 2016 14:31:59 +0000
Message-ID: <0000000@localhost/>
Content-Type: text/plain

Hi there :)`

	inbox.CreateMessage([]string{}, time.Now(), strings.NewReader(body))

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

var _ = Describe("Check", func() {

	var (
		server *server.Server
		input  = `{
	          "source": {
	            "imap" : {
	              "host": "localhost",
                      "port": "9875",
                      "username": "username",
                      "password": "password",
	              "inbox": "INBOX",
                      "skip_ssl_validation": true
	            }
	          }
	        }`
	)

	BeforeEach(func() {
		var err error
		server, err = startServer()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	It("works", func() {
		command := exec.Command(binaryPath)
		command.Stdin = strings.NewReader(input)

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, "10s").Should(gexec.Exit(0))

		output := session.Out.Contents()

		//memory backend implementation start at uid 6
		Expect(string(output)).Should(MatchJSON(`[{ "uid": 6 },{ "uid": 7 }]`))
	})
})
