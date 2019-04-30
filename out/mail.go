package out

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/domodwyer/mailyak"
	"github.com/pkg/errors"
)

//go:generate counterfeiter -o fakes/fake_mail.go types.go Mail
type Mail interface {
	From(string)
	To(...string)
	Cc(...string)
	Bcc(...string)
	Subject(string)
	AddHeader(name, value string)
	Attach(name string, r io.Reader)
	Plain() *mailyak.BodyPart
	HTML() *mailyak.BodyPart
	MimeBuf() (*bytes.Buffer, error)
}

type MailCreator struct {
	Mail                Mail
	From, Subject, Body string
	To, CC, BCC         []string
	headers             map[string]string
	attachments         map[string]io.Reader
	html                bool
}

func NewMailCreator() *MailCreator {
	return &MailCreator{
		Mail: mailyak.New("", nil),
	}
}
func (m *MailCreator) AddAttachment(filePath string) error {
	reader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	if m.attachments == nil {
		m.attachments = make(map[string]io.Reader)
	}
	m.attachments[filepath.Base(reader.Name())] = reader
	return nil
}

func (m *MailCreator) AddHeader(key, value string) {
	if m.headers == nil {
		m.headers = make(map[string]string)
	}
	if strings.EqualFold(key, "Content-Type") && strings.Contains(value, "text/html") {
		m.html = true
	}
	if strings.EqualFold(key, "MIME-version") || strings.EqualFold(key, "Content-Type") {
		return
	}
	m.headers[key] = value
}

func (m *MailCreator) Compose() ([]byte, error) {
	m.Mail.From(m.From)
	m.Mail.To(m.To...)
	m.Mail.Cc(m.CC...)
	m.Mail.Bcc(m.BCC...)
	m.Mail.Subject(m.Subject)
	if m.headers != nil {
		for key, value := range m.headers {
			m.Mail.AddHeader(key, value)
		}
	}
	if m.attachments != nil {
		for name, reader := range m.attachments {
			m.Mail.Attach(name, reader)
		}
	}
	if m.html {
		m.Mail.HTML().WriteString(m.Body)
	} else {
		m.Mail.Plain().WriteString(m.Body)
	}
	buf, err := m.Mail.MimeBuf()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get mime buffer")
	}
	return buf.Bytes(), nil
}
