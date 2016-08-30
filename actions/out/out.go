package out

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Input struct {
	Source struct {
		SMTP struct {
			Host     string
			Port     string
			Username string
			Password string
		}
		From string
		To   []string
	}
	Params struct {
		Subject       string
		BodyFile      string `json:"body_file"`
		Body          string
		SendEmptyBody bool `json:"send_empty_body"`
		Headers       string
	}
}

func Run(sourceRoot string, inBytes []byte) {
	if sourceRoot == "" {
		fmt.Fprintf(os.Stderr, "expected path to build sources as first argument")
		os.Exit(1)
	}

	inData := Input{}

	err := json.Unmarshal(inBytes, &inData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing input as JSON: %s", err)
		os.Exit(1)
	}

	if inData.Source.SMTP.Host == "" {
		fmt.Fprintf(os.Stderr, `missing required field "source.smtp.host"`)
		os.Exit(1)
	}

	if inData.Source.SMTP.Port == "" {
		fmt.Fprintf(os.Stderr, `missing required field "source.smtp.port"`)
		os.Exit(1)
	}

	if inData.Source.SMTP.Username != "" && inData.Source.SMTP.Password == "" {
		fmt.Fprintf(os.Stderr, `"source.smtp.password" is required, when "source.smtp.username" is given`)
		os.Exit(1)
	}

	if inData.Source.From == "" {
		fmt.Fprintf(os.Stderr, `missing required field "source.from"`)
		os.Exit(1)
	}

	if len(inData.Source.To) == 0 {
		fmt.Fprintf(os.Stderr, `missing required field "source.to"`)
		os.Exit(1)
	}

	if inData.Params.Subject == "" {
		fmt.Fprintf(os.Stderr, `missing required field "params.subject"`)
		os.Exit(1)
	}

	readSource := func(sourcePath string) (string, error) {
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(sourceRoot, sourcePath)
		}

		bytes, err := ioutil.ReadFile(sourcePath)
		return string(bytes), err
	}

	subject, err := readSource(inData.Params.Subject)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
	subject = strings.Trim(subject, "\n")

	var headers string
	if inData.Params.Headers != "" {
		headers, err = readSource(inData.Params.Headers)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(1)
		}
		headers = strings.Trim(headers, "\n")
	}

	body := inData.Params.Body

	if body == "" && inData.Params.BodyFile != "" {
		body, err = readSource(inData.Params.BodyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

	type MetadataItem struct {
		Name  string
		Value string
	}
	var outData struct {
		Version struct {
			Time time.Time
		} `json:"version"`
		Metadata []MetadataItem
	}
	outData.Version.Time = time.Now().UTC()
	outData.Metadata = []MetadataItem{
		{Name: "smtp_host", Value: inData.Source.SMTP.Host},
		{Name: "subject", Value: subject},
	}
	outBytes, err := json.Marshal(outData)
	if err != nil {
		panic(err)
	}

	var messageData []byte
	messageData = append(messageData, []byte("To: "+strings.Join(inData.Source.To, ", ")+"\n")...)
	if headers != "" {
		messageData = append(messageData, []byte(headers+"\n")...)
	}
	messageData = append(messageData, []byte("Subject: "+subject+"\n")...)

	messageData = append(messageData, []byte("\n")...)
	messageData = append(messageData, []byte(body)...)

	if inData.Params.SendEmptyBody == false && len(body) == 0 {
		fmt.Fprintf(os.Stderr, "Message not sent because the message body is empty and send_empty_body parameter was set to false. Github readme: https://github.com/pivotal-cf/email-resource")
		fmt.Printf("%s", []byte(outBytes))
		return
	}

	err = sendMail(inData, messageData)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to send an email using SMTP server %s on port %s: %v",
			inData.Source.SMTP.Host, inData.Source.SMTP.Port, err)
		os.Exit(1)
	}

	fmt.Printf("%s", []byte(outBytes))
}

func sendMail(inData Input, messageData []byte) error {
	if inData.Source.SMTP.Username == "" {
		return sendNoTlsNoAuthMail(inData, messageData)
	}

	return smtp.SendMail(
		fmt.Sprintf("%s:%s", inData.Source.SMTP.Host, inData.Source.SMTP.Port),
		smtp.PlainAuth(
			"",
			inData.Source.SMTP.Username,
			inData.Source.SMTP.Password,
			inData.Source.SMTP.Host,
		),
		inData.Source.From,
		inData.Source.To,
		messageData,
	)
}

func sendNoTlsNoAuthMail(inData Input, messageData []byte) error {
	client, err := smtp.Dial(fmt.Sprintf("%s:%s", inData.Source.SMTP.Host, inData.Source.SMTP.Port))
	if err != nil {
		return err
	}

	if err = client.Mail(inData.Source.From); err != nil {
		return err
	}

	for _, addr := range inData.Source.To {
		if err = client.Rcpt(addr); err != nil {
			return err
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}

	_, err = writer.Write(messageData)
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}
	client.Quit()

	return err
}
