package out

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

//Execute - provides out capability
func Execute(sourceRoot, version string, input []byte) (string, error) {

	logger := log.New(os.Stderr, "", log.LstdFlags)

	if sourceRoot == "" {
		return "", errors.New("expected path to build sources as first argument")
	}

	var indata Input

	err := json.Unmarshal(input, &indata)
	if err != nil {
		return "", err
	}

	err = validateConfiguration(indata)
	if err != nil {
		return "", errors.Wrap(err, "Invalid configuration")
	}

	source := indata.Source
	params := indata.Params
	smtpConfig := source.SMTP

	subject, err := fromTextOrFile(sourceRoot, params.SubjectText, params.Subject)
	if err != nil {
		return "", errors.Wrap(err, "Error getting Subject:")
	}
	subject = strings.Trim(subject, "\n")

	var headers string
	if params.Headers != "" {
		headers, err = readSource(sourceRoot, params.Headers)
		if err != nil {
			return "", errors.Wrap(err, "unable to read source file for headers")
		}
		headers = strings.Trim(headers, "\n")
	}

	body, err := fromTextOrFile(sourceRoot, params.BodyText, params.Body)
	if err != nil {
		return "", errors.Wrap(err, "Error getting Body:")
	}

	if params.To != "" {
		var toList string
		toList, err = readSource(sourceRoot, params.To)
		if err != nil {
			return "", errors.Wrap(err, "Error getting To:")
		}
		if len(toList) > 0 {
			toListArray := strings.Split(toList, ",")
			for _, toAddress := range toListArray {
				source.To = append(source.To, strings.TrimSpace(toAddress))
			}
		}
	}

	if params.Bcc != "" {
		var bccList string
		bccList, err = readSource(sourceRoot, params.Bcc)
		if err != nil {
			return "", errors.Wrap(err, "Error getting BCC:")
		}
		if len(bccList) > 0 {
			bccListArray := strings.Split(bccList, ",")
			for _, bccAddress := range bccListArray {
				source.Bcc = append(source.Bcc, strings.TrimSpace(bccAddress))
			}
		}
	}

	var outdata Output
	outdata.Version.Time = time.Now().UTC()
	outdata.Metadata = []MetadataItem{
		{Name: "smtp_host", Value: smtpConfig.Host},
		{Name: "subject", Value: subject},
		{Name: "version", Value: version},
	}
	outbytes, err := json.Marshal(outdata)
	if err != nil {
		return "", errors.Wrap(err, "Error Marshalling JSON:")
	}

	if params.SendEmptyBody == false && len(body) == 0 {
		logger.Println("Message not sent because the message body is empty and send_empty_body parameter was set to false. Github readme: https://github.com/pivotal-cf/email-resource")
		return string(outbytes), nil
	}
	var messageData []byte
	messageData = append(messageData, []byte("To: "+strings.Join(source.To, ", ")+"\n")...)
	messageData = append(messageData, []byte("From: "+source.From+"\n")...)
	if headers != "" {
		messageData = append(messageData, []byte(headers+"\n")...)
	}
	messageData = append(messageData, []byte("Subject: "+subject+"\n")...)

	messageData = append(messageData, []byte("\n")...)
	messageData = append(messageData, []byte(body)...)

	var c *smtp.Client
	var wc io.WriteCloser
	c, err = smtp.Dial(fmt.Sprintf("%s:%s", smtpConfig.Host, smtpConfig.Port))
	if err != nil {
		return "", err
	}
	defer c.Close()

	hostOrigin := "localhost"

	if smtpConfig.HostOrigin != "" {
		hostOrigin = smtpConfig.HostOrigin
	}
	if err = c.Hello(hostOrigin); err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("unable to connect with hello with host name %s, try setting property host_origin", hostOrigin))
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		config := tlsConfig(smtpConfig)

		if err = c.StartTLS(config); err != nil {
			return "", errors.Wrap(err, "unable to start TLS")
		}
	}

	err = doAuth(smtpConfig, c)
	if err != nil {
		return "", errors.Wrap(err, "Error doing auth:")
	}
	if err = c.Mail(source.From); err != nil {
		return "", errors.Wrap(err, "Error setting from:")
	}
	for _, addr := range source.To {
		if err = c.Rcpt(addr); err != nil {
			return "", errors.Wrap(err, "Error setting to:")
		}
	}
	for _, addr := range source.Bcc {
		if err = c.Rcpt(addr); err != nil {
			return "", errors.Wrap(err, "Error setting bcc:")
		}
	}
	wc, err = c.Data()
	if err != nil {
		return "", errors.Wrap(err, "Error getting Data:")
	}
	_, err = wc.Write(messageData)
	if err != nil {
		return "", errors.Wrap(err, "Error writting message data:")
	}
	err = wc.Close()
	if err != nil {
		return "", errors.Wrap(err, "Error closing:")
	}
	err = c.Quit()
	if err != nil {
		return "", errors.Wrap(err, "Error quitting:")
	}

	return string(outbytes), err
}

func validateConfiguration(indata Input) error {
	if indata.Source.SMTP.Host == "" {
		return errors.New(`missing required field "source.smtp.host"`)
	}

	if indata.Source.SMTP.Port == "" {
		return errors.New(`missing required field "source.smtp.port"`)
	}

	if indata.Source.From == "" {
		return errors.New(`missing required field "source.from"`)
	}

	if len(indata.Source.To) == 0 && len(indata.Params.To) == 0 {
		return errors.New(`missing required field "source.to" or "params.to". Must specify at least one`)
	}

	if indata.Params.Subject == "" && indata.Params.SubjectText == "" {
		return errors.New(`missing required field "params.subject" or "params.subject_text". Must specify at least one`)
	}

	if indata.Source.SMTP.Anonymous == false {
		if indata.Source.SMTP.Username == "" {
			return errors.New(`missing required field "source.smtp.username" if anonymous specify anonymous: true`)
		}

		if indata.Source.SMTP.Password == "" {
			return errors.New(`missing required field "source.smtp.password" if anonymous specify anonymous: true`)
		}
	}
	return nil
}

func doAuth(smtpConfig SMTP, c *smtp.Client) error {
	if smtpConfig.Anonymous {
		return nil
	}
	if smtpConfig.LoginAuth {
		auth := LoginAuth(smtpConfig.Username, smtpConfig.Password)

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
			smtpConfig.Username,
			smtpConfig.Password,
			smtpConfig.Host,
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

func tlsConfig(smtpConfig SMTP) *tls.Config {
	config := &tls.Config{
		ServerName: smtpConfig.Host,
	}

	if smtpConfig.SkipSSLValidation {
		config.InsecureSkipVerify = smtpConfig.SkipSSLValidation
		return config
	}

	if smtpConfig.CaCert != "" {
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM([]byte(smtpConfig.CaCert))

		config.RootCAs = caPool

		return config
	}

	return config
}

func replaceTokens(sourceString string) string {
	var buildTokens = map[string]string{
		"${BUILD_ID}":            os.Getenv("BUILD_ID"),
		"${BUILD_NAME}":          os.Getenv("BUILD_NAME"),
		"${BUILD_JOB_NAME}":      os.Getenv("BUILD_JOB_NAME"),
		"${BUILD_PIPELINE_NAME}": os.Getenv("BUILD_PIPELINE_NAME"),
		"${ATC_EXTERNAL_URL}":    os.Getenv("ATC_EXTERNAL_URL"),
		"${BUILD_TEAM_NAME}":     os.Getenv("BUILD_TEAM_NAME"),
	}
	for k, v := range buildTokens {
		sourceString = strings.Replace(sourceString, k, v, -1)
	}
	return sourceString
}

func readSource(sourceRoot, sourcePath string) (string, error) {
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(sourceRoot, sourcePath)
	}
	bytes, err := ioutil.ReadFile(sourcePath)
	return replaceTokens(string(bytes)), err
}

func fromTextOrFile(sourceRoot, text, filePath string) (string, error) {
	if text != "" {
		return replaceTokens(text), nil

	}
	if filePath != "" {
		return readSource(sourceRoot, filePath)
	}
	return "", nil
}
