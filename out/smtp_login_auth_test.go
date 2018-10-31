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
	return &emailSender{conf: conf, send: smtp.SendMail}
}

type emailSender struct {
	conf EmailConfig
	auth smtp.Auth
	send func(string, smtp.Auth, string, []string, []byte) error
}

func (e *emailSender) Send(to []string, body []byte) error {
	addr := e.conf.ServerHost + ":" + e.conf.ServerPort
	auth := out.LoginAuth(e.conf.Username, e.conf.Password)
	e.auth = auth
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

func TestStart(t *testing.T) {
	username := "test-user"
	password := "test-pass"
	auth := out.LoginAuth(username, password)

	authType, resp, err := auth.Start(nil)

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if authType != "LOGIN" {
		t.Errorf("expected authType to be LOGIN, but was %s", authType)
	}
	if string(resp) != "" {
		t.Errorf("wrong message body.\n\nexpected: %s\n got: %s", "", string(resp))
	}
}

func TestNext(t *testing.T) {
	username := "test-user"
	password := "test-pass"

	auth := out.LoginAuth(username, password)

	resp, err := auth.Next([]byte("Username:"), true)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if string(resp) != username {
		t.Errorf("expected username response to be %s, got %s", username, string(resp))
	}

	resp, err = auth.Next([]byte("Password:"), true)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if string(resp) != password {
		t.Errorf("expected password response to be %s, got %s", password, string(resp))
	}
}
