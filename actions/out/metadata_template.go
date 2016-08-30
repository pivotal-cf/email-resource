package out

import (
	"text/template"
	"bytes"
	"os"
)

func RenderTemplate(text string) string {
	tmpl := template.New("Metadata Template").Delims("${", "}").Funcs(template.FuncMap{
		"BUILD_ID": makeTemplateFunc("BUILD_ID"),
		"BUILD_NAME": makeTemplateFunc("BUILD_NAME"),
		"BUILD_JOB_NAME": makeTemplateFunc("BUILD_JOB_NAME"),
		"BUILD_PIPELINE_NAME": makeTemplateFunc("BUILD_PIPELINE_NAME"),
		"ATC_EXTERNAL_URL": makeTemplateFunc("ATC_EXTERNAL_URL"),
	})
	parsedTemplate, err := tmpl.Parse(text)

	if err != nil {
		return text
	}

	output := new(bytes.Buffer)
	err = parsedTemplate.Execute(output, nil)
	if err != nil {
		return text
	}
	return output.String()
}

func makeTemplateFunc(envVar string) func() string {
	return func() string {
		return os.Getenv(envVar)
	}
}