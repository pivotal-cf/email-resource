package out

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
)

//Execute - provides out capability
func Execute(sourceRoot, version string, input []byte) (string, error) {
	var buildTokens = map[string]string{
		"${BUILD_ID}":            os.Getenv("BUILD_ID"),
		"${BUILD_NAME}":          os.Getenv("BUILD_NAME"),
		"${BUILD_JOB_NAME}":      os.Getenv("BUILD_JOB_NAME"),
		"${BUILD_PIPELINE_NAME}": os.Getenv("BUILD_PIPELINE_NAME"),
		"${ATC_EXTERNAL_URL}":    os.Getenv("ATC_EXTERNAL_URL"),
		"${BUILD_TEAM_NAME}":     os.Getenv("BUILD_TEAM_NAME"),
	}

	if sourceRoot == "" {
		return "", errors.New("expected path to build sources as first argument")
	}

	var indata Input

	err := json.Unmarshal(input, &indata)
	if err != nil {
		return "", err
	}

	if indata.Source.SMTP.Host == "" {
		return "", errors.New(`missing required field "source.smtp.host"`)
	}

	if indata.Source.SMTP.Port == "" {
		return "", errors.New(`missing required field "source.smtp.port"`)
	}

	if indata.Source.From == "" {
		return "", errors.New(`missing required field "source.from"`)
	}

	if len(indata.Source.To) == 0 && len(indata.Params.To) == 0 {
		return "", errors.New(`missing required field "source.to" or "params.to". Must specify at least one`)
	}

	if indata.Params.Subject == "" {
		return "", errors.New(`missing required field "params.subject"`)
	}

	if indata.Source.SMTP.Anonymous == false {
		if indata.Source.SMTP.Username == "" {
			return "", errors.New(`missing required field "source.smtp.username" if anonymous specify anonymous: true`)
		}

		if indata.Source.SMTP.Password == "" {
			return "", errors.New(`missing required field "source.smtp.password" if anonymous specify anonymous: true`)
		}
	}

	replaceTokens := func(sourceString string) string {
		for k, v := range buildTokens {
			sourceString = strings.Replace(sourceString, k, v, -1)
		}
		return sourceString
	}

	readSource := func(sourcePath string) (string, error) {
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(sourceRoot, sourcePath)
		}
		var bytes []byte
		bytes, err = ioutil.ReadFile(sourcePath)
		return replaceTokens(string(bytes)), err
	}

	subject, err := readSource(indata.Params.Subject)
	if err != nil {
		return "", err
	}
	subject = strings.Trim(subject, "\n")

	contentType := "text/plain"
	headerMap := make(map[string][]string)
	if indata.Params.Headers != "" {
		var headers string
		headers, err = readSource(indata.Params.Headers)
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(strings.Trim(headers, "\n"), "\n") {
			header := strings.Split(line, ": ")
			headerKey := strings.TrimSpace(header[0])
			if headerKey == "Content-Type" {
				contentType = strings.Split(header[1], ";")[0]
			} else if strings.ToUpper(headerKey) == "MIME-VERSION" {
				//do nothing....
			} else {
				headerMap[strings.TrimSpace(header[0])] = header[1:]
			}
		}

	}

	var body string
	if indata.Params.Body != "" {
		body, err = readSource(indata.Params.Body)
		if err != nil {
			return "", err
		}
	}

	if indata.Params.To != "" {
		var toList string
		toList, err = readSource(indata.Params.To)
		if err != nil {
			return "", err
		}
		if len(toList) > 0 {
			toListArray := strings.Split(toList, ",")
			for _, toAddress := range toListArray {
				indata.Source.To = append(indata.Source.To, strings.TrimSpace(toAddress))
			}
		}
	}

	var outdata Output
	outdata.Version.Time = time.Now().UTC()
	outdata.Metadata = []MetadataItem{
		{Name: "smtp_host", Value: indata.Source.SMTP.Host},
		{Name: "subject", Value: subject},
		{Name: "version", Value: version},
	}
	outbytes, err := json.Marshal(outdata)
	if err != nil {
		return "", err
	}

	if indata.Params.SendEmptyBody == false && len(body) == 0 {
		fmt.Fprintf(os.Stderr, "Message not sent because the message body is empty and send_empty_body parameter was set to false. Github readme: https://github.com/pivotal-cf/email-resource")
		fmt.Printf("%s", []byte(outbytes))
		return string(outbytes), nil
	}
	m := gomail.NewMessage()
	m.SetHeader("From", indata.Source.From)
	m.SetHeader("To", indata.Source.To...)
	if len(headerMap) > 0 {
		m.SetHeaders(headerMap)
	}
	m.SetHeader("Subject", subject)
	m.SetBody(contentType, body)

	var dialer *gomail.Dialer
	port, err := strconv.Atoi(indata.Source.SMTP.Port)
	if indata.Source.SMTP.Anonymous {
		dialer = &gomail.Dialer{Host: indata.Source.SMTP.Host, Port: port}
	} else {
		dialer = gomail.NewDialer(indata.Source.SMTP.Host, port, indata.Source.SMTP.Username, indata.Source.SMTP.Password)
	}
	if indata.Source.SMTP.SkipSSLValidation {
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: indata.Source.SMTP.SkipSSLValidation}
	}
	if err = dialer.DialAndSend(m); err != nil {
		return "", err
	}

	return string(outbytes), nil
}
