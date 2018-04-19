package in

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/pivotal-cf/email-resource/check"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type MetadataItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Attachment struct {
	Name     string
	Contents []byte
}

//Execute - provides in capability
func Execute(input check.IMAP, version check.Version, destinationDir string) (string, error) {
	err := os.MkdirAll(destinationDir, os.ModePerm)
	if err != nil {
		return "", err
	}

	imapClient, err := client.DialTLS(input.Host+":"+input.Port, &tls.Config{
		InsecureSkipVerify: input.SkipSSLValidation,
	})
	if err != nil {
		return "", err
	}

	defer imapClient.Logout()

	if err := imapClient.Login(input.Username, input.Password); err != nil {
		return "", err
	}

	_, err = imapClient.Select(input.Inbox, true)
	if err != nil {
		return "", err
	}

	v, err := strconv.Atoi(version.ID)
	if err != nil {
		return "", err
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uint32(v))

	messages, done := fetchMessages(imapClient, seqset)
	msg := <-messages
	if msg == nil {
		return "", errors.New("server didn't return message")
	}

	if err := <-done; err != nil {
		return "", err
	}

	section := &imap.BodySectionName{}
	messageBodyReader := msg.GetBody(section)
	if messageBodyReader == nil {
		return "", errors.New("server didn't return message body")
	}

	mailReader, err := mail.CreateReader(messageBodyReader)
	if err != nil {
		return "", err
	}

	var body []byte
	var attachments []Attachment

	for {
		part, err := mailReader.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}

		switch header := part.Header.(type) {
		case mail.TextHeader:
			body, _ = ioutil.ReadAll(part.Body)
		case mail.AttachmentHeader:
			filename, _ := header.Filename()
			contents, _ := ioutil.ReadAll(part.Body)
			attachments = append(attachments, Attachment{
				Name:     filename,
				Contents: contents,
			})
		}
	}

	ioutil.WriteFile(filepath.Join(destinationDir, "version"), []byte(msg.Envelope.MessageId), 0600)
	ioutil.WriteFile(filepath.Join(destinationDir, "subject"), []byte(msg.Envelope.Subject), 0600)
	ioutil.WriteFile(filepath.Join(destinationDir, "date"), []byte(msg.Envelope.Date.Format(time.RFC850)), 0600)
	ioutil.WriteFile(filepath.Join(destinationDir, "body"), body, 0600)

	attachmentsDir := filepath.Join(destinationDir, "attachments")
	os.MkdirAll(attachmentsDir, os.ModePerm)
	for _, attachment := range attachments {
		ioutil.WriteFile(filepath.Join(attachmentsDir, attachment.Name), attachment.Contents, 0600)
	}

	var data struct {
		Version  check.Version  `json:"version"`
		Metadata []MetadataItem `json:"metadata"`
	}

	data.Version = version
	data.Metadata = append(data.Metadata,
		MetadataItem{Name: "Subject", Value: msg.Envelope.Subject},
		MetadataItem{Name: "Date", Value: msg.Envelope.Date.Format(time.RFC850)},
		MetadataItem{Name: "Number of Attachments", Value: strconv.Itoa(len(attachments))},
	)

	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func fetchMessages(imapClient *client.Client, seqset *imap.SeqSet) (chan *imap.Message, chan error) {
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchUid, imap.FetchEnvelope, section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- imapClient.UidFetch(seqset, items, messages)
	}()
	return messages, done
}
