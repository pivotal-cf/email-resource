package out

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func NewSender(host, port, username, password string, debug bool, logger *log.Logger) *Sender {
	return &Sender{
		host:        host,
		port:        port,
		attachments: make(map[string]io.Reader),
		debug:       debug,
		logger:      logger,
		username:    username,
		password:    password,
	}
}

type Sender struct {
	host                                    string
	port                                    string
	attachments                             map[string]io.Reader
	debug                                   bool
	logger                                  *log.Logger
	HostOrigin                              string
	CaCert                                  string
	Anonymous, LoginAuth, SkipSSLValidation bool
	username                                string
	password                                string
	From                                    string
	To                                      []string
}

func (s *Sender) AddAttachment(filePath string) error {
	reader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	s.attachments[filepath.Base(reader.Name())] = reader
	return nil
}

func (s *Sender) Send(msg []byte) error {
	var c *smtp.Client
	var err error
	var wc io.WriteCloser
	if s.debug {
		s.logger.Println("Dialing")
	}
	c, err = smtp.Dial(fmt.Sprintf("%s:%s", s.host, s.port))
	if err != nil {
		return errors.Wrap(err, "Error Dialing smtp server")
	}
	defer c.Close()

	hostOrigin := "localhost"

	if s.HostOrigin != "" {
		hostOrigin = s.HostOrigin
	}
	if s.debug {
		s.logger.Println("Saying Hello to SMTP Server")
	}
	if err = c.Hello(hostOrigin); err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to connect with hello with host name %s, try setting property host_origin", hostOrigin))
	}
	if s.debug {
		s.logger.Println("STARTTLS with SMTP Server")
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		config := s.tlsConfig()

		if err = c.StartTLS(config); err != nil {
			return errors.Wrap(err, "unable to start TLS")
		}
	}

	if s.debug {
		s.logger.Println("Authenticating with SMTP Server")
	}
	err = s.doAuth(c)
	if err != nil {
		return errors.Wrap(err, "Error doing auth:")
	}
	if s.debug {
		s.logger.Println("Setting From")
	}
	if err = c.Mail(s.From); err != nil {
		return errors.Wrap(err, "Error setting from:")
	}
	if s.debug {
		s.logger.Println("Setting TO")
	}
	for _, addr := range s.To {
		if err = c.Rcpt(addr); err != nil {
			if errCode, ok := err.(*textproto.Error); ok && errCode.Code == 550 {
				s.logger.Printf("Skipping %s: %s\n", addr, err.Error())
				continue
			}
			return errors.Wrap(err, "Error setting to:")
		}
	}

	if s.debug {
		s.logger.Println("Getting Data from SMTP Server")
	}
	wc, err = c.Data()
	if err != nil {
		return errors.Wrap(err, "Error getting Data:")
	}
	if s.debug {
		s.logger.Println(fmt.Sprintf("Writing message to SMTP Server %s", string(msg)))
	}
	_, err = wc.Write(msg)
	if err != nil {
		return errors.Wrap(err, "Error writting message data:")
	}
	if s.debug {
		s.logger.Println("Closing connection to SMTP Server")
	}
	err = wc.Close()
	if err != nil {
		return errors.Wrap(err, "Error closing:")
	}
	if s.debug {
		s.logger.Println("Quitting connection to SMTP Server")
	}
	err = c.Quit()
	if err != nil {
		return errors.Wrap(err, "Error quitting:")
	}
	return nil
}

func (s *Sender) tlsConfig() *tls.Config {
	config := &tls.Config{
		ServerName: s.host,
	}

	if s.SkipSSLValidation {
		config.InsecureSkipVerify = s.SkipSSLValidation
		return config
	}

	if s.CaCert != "" {
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM([]byte(s.CaCert))
		config.RootCAs = caPool
		return config
	}

	return config
}

func (s *Sender) doAuth(c *smtp.Client) error {
	if s.Anonymous {
		return nil
	}
	if s.LoginAuth {
		auth := LoginAuth(s.username, s.password)

		if auth != nil {
			if ok, _ := c.Extension("AUTH"); ok {
				if err := c.Auth(auth); err != nil {
					return errors.Wrap(err, "unable to auth using type Login Auth")
				}
			}
		}
	} else {
		auth := smtp.PlainAuth(
			"",
			s.username,
			s.password,
			s.host,
		)
		if auth != nil {
			if ok, _ := c.Extension("AUTH"); ok {
				if err := c.Auth(auth); err != nil {
					return errors.Wrap(err, "unable to auth using type Plain Auth")
				}
			}
		}
	}
	return nil
}
