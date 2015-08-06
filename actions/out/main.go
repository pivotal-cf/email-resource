package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/smtp"
	"os"
	"path/filepath"
	"time"
)

func main() {
	sourceRoot := os.Args[1]
	if sourceRoot == "" {
		fmt.Fprintf(os.Stderr, "expected path to build sources as first argument")
		os.Exit(1)
	}

	var indata struct {
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
			Subject string
			Body    string
		}
	}

	inbytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(inbytes, &indata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing input as JSON: %s", err)
		os.Exit(1)
	}

	if indata.Source.SMTP.Host == "" {
		fmt.Fprintf(os.Stderr, `missing required field "source.smtp.host"`)
		os.Exit(1)
	}

	if indata.Source.SMTP.Port == "" {
		fmt.Fprintf(os.Stderr, `missing required field "source.smtp.port"`)
		os.Exit(1)
	}

	if indata.Source.SMTP.Username == "" {
		fmt.Fprintf(os.Stderr, `missing required field "source.smtp.username"`)
		os.Exit(1)
	}

	if indata.Source.SMTP.Password == "" {
		fmt.Fprintf(os.Stderr, `missing required field "source.smtp.password"`)
		os.Exit(1)
	}

	if indata.Source.From == "" {
		fmt.Fprintf(os.Stderr, `missing required field "source.from"`)
		os.Exit(1)
	}

	if len(indata.Source.To) == 0 {
		fmt.Fprintf(os.Stderr, `missing required field "source.to"`)
		os.Exit(1)
	}

	if indata.Params.Subject == "" {
		fmt.Fprintf(os.Stderr, `missing required field "params.subject"`)
		os.Exit(1)
	}

	readSource := func(sourcePath string) ([]byte, error) {
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(sourceRoot, sourcePath)
		}

		return ioutil.ReadFile(sourcePath)
	}

	subjectBytes, err := readSource(indata.Params.Subject)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	var bodyBytes []byte
	if indata.Params.Body != "" {
		bodyBytes, err = readSource(indata.Params.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

	type MetadataItem struct {
		Name  string
		Value string
	}
	var outdata struct {
		Version struct {
			Time time.Time
		} `json:"version"`
		Metadata []MetadataItem
	}
	outdata.Version.Time = time.Now().UTC()
	outdata.Metadata = []MetadataItem{
		{Name: "smtp_host", Value: indata.Source.SMTP.Host},
		{Name: "subject", Value: string(subjectBytes)},
	}
	outbytes, err := json.Marshal(outdata)
	if err != nil {
		panic(err)
	}

	messageData := []byte("Subject: " + string(subjectBytes) + "\n")
	messageData = append(messageData, bodyBytes...)

	err = smtp.SendMail(
		fmt.Sprintf("%s:%s", indata.Source.SMTP.Host, indata.Source.SMTP.Port),
		smtp.PlainAuth(
			"",
			indata.Source.SMTP.Username,
			indata.Source.SMTP.Password,
			indata.Source.SMTP.Host,
		),
		indata.Source.From,
		indata.Source.To,
		messageData,
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", []byte(outbytes))
}
