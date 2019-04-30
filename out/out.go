package out

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
		return "", errors.Wrap(err, "unmarshalling input")
	}

	err = validateConfiguration(indata)
	if err != nil {
		return "", errors.Wrap(err, "Invalid configuration")
	}

	source := indata.Source
	params := indata.Params
	debug := strings.EqualFold("true", params.Debug)
	smtpConfig := source.SMTP

	if debug {
		logger.Println("Getting subject")
	}
	subject, err := fromTextOrFile(sourceRoot, params.SubjectText, params.Subject)
	if err != nil {
		return "", errors.Wrap(err, "Error getting Subject:")
	}
	subject = strings.Trim(subject, "\n")

	headers := make(map[string]string)
	if params.Headers != "" {
		if debug {
			logger.Println("Getting headers")
		}
		var headersString string
		headersString, err = readSource(sourceRoot, params.Headers)
		if err != nil {
			return "", errors.Wrap(err, "unable to read source file for headers")
		}
		headersString = strings.Trim(headersString, "\n")
		lines := strings.Split(headersString, "\n")
		for _, line := range lines {
			kv := strings.Split(line, ": ")
			headers[kv[0]] = kv[1]
		}
	}

	if debug {
		logger.Println("Getting Body")
	}
	body, err := fromTextOrFile(sourceRoot, params.BodyText, params.Body)
	if err != nil {
		return "", errors.Wrap(err, "Error getting Body:")
	}

	if params.To != "" {
		if debug {
			logger.Println("Getting To Params")
		}
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

	if params.Cc != "" {
		if debug {
			logger.Println("Getting CC Params")
		}
		var ccList string
		ccList, err = readSource(sourceRoot, params.Cc)
		if err != nil {
			return "", errors.Wrap(err, "Error getting CC:")
		}
		if len(ccList) > 0 {
			ccListArray := strings.Split(ccList, ",")
			for _, ccAddress := range ccListArray {
				source.Cc = append(source.Cc, strings.TrimSpace(ccAddress))
			}
		}
	}

	if params.Bcc != "" {
		if debug {
			logger.Println("Getting BCC Params")
		}
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

	if debug {
		logger.Println("Building Message Payload")
	}
	sender := NewSender(smtpConfig.Host, smtpConfig.Port, debug, logger)
	sender.HostOrigin = smtpConfig.HostOrigin
	sender.CaCert = smtpConfig.CaCert
	sender.Anonymous = smtpConfig.Anonymous
	sender.LoginAuth = smtpConfig.LoginAuth
	sender.SkipSSLValidation = smtpConfig.SkipSSLValidation
	sender.Username = smtpConfig.Username
	sender.Password = smtpConfig.Password
	sender.From = source.From
	sender.To = source.To
	sender.Cc = source.Cc
	sender.Bcc = source.Bcc
	sender.Subject = subject
	sender.Body = body
	sender.Headers = headers
	if len(params.AttachmentGlobs) > 0 {
		for _, glob := range params.AttachmentGlobs {
			logger.Println(fmt.Sprintf("Looking for files with pattern %s", glob))
			paths, err := filepath.Glob(glob)
			if err != nil {
				return "", errors.Wrapf(err, "Error getting files from glob %s", glob)
			}
			for _, attachmentPath := range paths {
				logger.Println(fmt.Sprintf("Attaching files %s", attachmentPath))
				err = sender.AddAttachment(attachmentPath)
				if err != nil {
					return "", errors.Wrapf(err, "Error adding attachement from path %s", attachmentPath)
				}
			}
		}
	}
	err = sender.Send()

	if err != nil {
		return "", err
	}

	return string(outbytes), nil
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
