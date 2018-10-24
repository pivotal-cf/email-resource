package out_test

import (
	"net/smtp"
	"testing"

	"github.com/pivotal-cf/email-resource/out"
)

type EmailConfig struct {
	Username   string
	Password   string
	ServerHost string
	ServerPort string
	SenderAddr string
}

type EmailSender interface {
	Send(to []string, body []byte) error
}

func NewEmailSender(conf EmailConfig) EmailSender {
	return &emailSender{conf, smtp.SendMail}
}

type emailSender struct {
	conf EmailConfig
	send func(string, smtp.Auth, string, []string, []byte) error
}

func (e *emailSender) Send(to []string, body []byte) error {
	addr := e.conf.ServerHost + ":" + e.conf.ServerPort
	auth := out.LoginAuth(e.conf.Username, e.conf.Password)
	return e.send(addr, auth, e.conf.SenderAddr, to, body)
}

/****** testing ******/

func TestEmail_SendSuccessful(t *testing.T) {
	f, r := mockSend(nil)
	sender := &emailSender{send: f}
	body := "Hello World"
	err := sender.Send([]string{"me@example.com"}, []byte(body))

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if string(r.msg) != body {
		t.Errorf("wrong message body.\n\nexpected: %s\n got: %s", body, r.msg)
	}
}

func mockSend(errToReturn error) (func(string, smtp.Auth, string, []string, []byte) error, *emailRecorder) {
	r := new(emailRecorder)
	return func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		*r = emailRecorder{addr, a, from, to, msg}
		return errToReturn
	}, r
}

type emailRecorder struct {
	addr string
	auth smtp.Auth
	from string
	to   []string
	msg  []byte
}
