package check

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type IMAP struct {
	Host              string `json:"host"`
	Port              string `json:"port"`
	Username          string `json:"username"`
	Password          string `json:"password"`
	Inbox             string `json:"inbox"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"`
}

type Version struct {
	ID   string `json:"uid"`
}

//Execute - provides check capability
func Execute(input IMAP) (string, error) {
	imapClient, err := client.DialTLS(input.Host+":"+input.Port, &tls.Config{
		InsecureSkipVerify: input.SkipSSLValidation,
	})
	if err != nil {
		return "", fmt.Errorf("connecting to server; %s", err)
	}

	defer imapClient.Logout()

	if err := imapClient.Login(input.Username, input.Password); err != nil {
		return "", err
	}

	mbox, err := imapClient.Select(input.Inbox, true)
	if err != nil {
		return "", err
	}

	seqset := generateSeqset(mbox)

	messages, done := fetchMessages(imapClient, seqset)

	results, err := retrieveVersions(messages, done)
	if err != nil {
		return "", err
	}

	contents, err := json.Marshal(results)
	if err != nil {
		return "", err
	}

	return string(contents), nil
}

func generateSeqset(mbox *imap.MailboxStatus) *imap.SeqSet {
	var (
		numberOfMessagesToReadFrom = uint32(5)
		from                       = mbox.Messages
		to                         uint32
	)

	if mbox.Messages > numberOfMessagesToReadFrom {
		to = mbox.Messages - numberOfMessagesToReadFrom
	} else {
		to = mbox.Messages - uint32(1)
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(to, from)
	return seqset
}

func fetchMessages(imapClient *client.Client, seqset *imap.SeqSet) (chan *imap.Message, chan error) {
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- imapClient.Fetch(seqset, []imap.FetchItem{imap.FetchUid}, messages)
	}()
	return messages, done
}

func retrieveVersions(messages chan *imap.Message, done chan error) ([]Version, error) {
	var results []Version

	for {
		select {
		case msg := <-messages:
			if msg == nil {
				continue
			}

			results = append(results, Version{
				ID:   strconv.Itoa(int(msg.Uid)),
			})
		case err := <-done:
			if err != nil {
				return nil, err
			}

			return results, nil
		}
	}
}
