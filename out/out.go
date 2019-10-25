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
		logger.Println(fmt.Sprintf("Params: %+v", params))
	}
	if debug {
		logger.Println("Getting subject")
	}
	subject, err := fromTextOrFile(sourceRoot, params.SubjectText, params.Subject)
	if err != nil {
		return "", errors.Wrap(err, "Error getting Subject:")
	}
	subject = strings.Trim(subject, "\n")

	if debug {
		logger.Println("Getting Body")
	}
	body, err := fromTextOrFile(sourceRoot, params.BodyText, params.Body)
	if err != nil {
		return "", errors.Wrap(err, "Error getting Body:")
	}

	toArray, err := sliceFromTextOrFile(sourceRoot, params.ToText, params.To)
	if err != nil {
		return "", errors.Wrap(err, "Error getting to list:")
	}
	source.To = append(source.To, toArray...)

	ccArray, err := sliceFromTextOrFile(sourceRoot, params.CcText, params.Cc)
	if err != nil {
		return "", errors.Wrap(err, "Error getting cc list:")
	}
	source.Cc = append(source.Bcc, ccArray...)

	bccArray, err := sliceFromTextOrFile(sourceRoot, params.BccText, params.Bcc)
	if err != nil {
		return "", errors.Wrap(err, "Error getting cc list:")
	}
	source.Bcc = append(source.Bcc, bccArray...)

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

	mail := NewMailCreator()
	mail.From = source.From
	mail.To = source.To
	mail.CC = source.Cc
	mail.BCC = source.Bcc
	mail.Subject = subject
	mail.Body = body
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
			mail.AddHeader(kv[0], kv[1])
		}
	}

	if len(params.AttachmentGlobs) > 0 {
		for _, glob := range params.AttachmentGlobs {
			globPath := filepath.Join(sourceRoot, glob)
			logger.Println(fmt.Sprintf("Looking for files with pattern %s", globPath))
			paths, err := filepath.Glob(globPath)
			if err != nil {
				return "", errors.Wrapf(err, "Error getting files from glob %s", globPath)
			}
			for _, attachmentPath := range paths {
				logger.Println(fmt.Sprintf("Attaching files %s", attachmentPath))
				err = mail.AddAttachment(attachmentPath)
				if err != nil {
					return "", errors.Wrapf(err, "Error adding attachement from path %s", attachmentPath)
				}
			}
		}
	}

	sender := NewSender(smtpConfig.Host, smtpConfig.Port, smtpConfig.Username, smtpConfig.Password, debug, logger)
	sender.HostOrigin = smtpConfig.HostOrigin
	sender.CaCert = smtpConfig.CaCert
	sender.Anonymous = smtpConfig.Anonymous
	sender.LoginAuth = smtpConfig.LoginAuth
	sender.SkipSSLValidation = smtpConfig.SkipSSLValidation
	sender.From = source.From
	sender.To = append(append(source.To, source.Cc...), source.Bcc...)

	msg, err := mail.Compose()
	if err != nil {
		return "", errors.Wrapf(err, "Error composing mail")
	}
	err = sender.Send(msg)
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

	if len(indata.Source.To) == 0 && len(indata.Params.To) == 0 && len(indata.Params.ToText) == 0 {
		return errors.New(`missing required field "source.to" or "params.to" or "params.to_text". Must specify at least one`)
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

func sliceFromTextOrFile(sourceRoot, text, filePath string) ([]string, error) {
	var returnList []string
	if text != "" {
		listArray := strings.Split(text, ",")
		for _, item := range listArray {
			returnList = append(returnList, strings.TrimSpace(item))
		}
	}
	if filePath != "" {
		fileList, err := readSource(sourceRoot, filePath)
		if err != nil {
			return nil, errors.Wrapf(err, "Error reading file %s", filePath)
		}
		if len(fileList) > 0 {
			listArray := strings.Split(fileList, ",")
			for _, item := range listArray {
				returnList = append(returnList, strings.TrimSpace(item))
			}
		}
	}
	return returnList, nil
}
