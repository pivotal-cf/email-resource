package out

import "time"

//Input - Struct that represents the input to out
type Input struct {
	Source Source `json:"source"`
	Params Params `json:"params"`
}

type Source struct {
	SMTP SMTP `json:"smtp"`
	From string
	To   []string
	Bcc  []string
}

type Params struct {
	Subject       string
	SubjectText   string `json:"subject_text"`
	Body          string
	BodyText      string `json:"body_text"`
	SendEmptyBody bool   `json:"send_empty_body"`
	Headers       string
	HeadersText   string `json:"headers_text"`
	To            string `json:"to"`
	Bcc           string `json:"bcc"`
	Debug         string `json:"debug"`
}

type SMTP struct {
	Host              string
	Port              string
	Username          string
	Password          string
	Anonymous         bool   `json:"anonymous"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"`
	CaCert            string `json:"ca_cert"`
	HostOrigin        string `json:"host_origin"`
	LoginAuth         bool   `json:"login_auth"`
}

//MetadataItem - metadata within output
type MetadataItem struct {
	Name  string
	Value string
}

//Output - represents output from out
type Output struct {
	Version struct {
		Time time.Time
	} `json:"version"`
	Metadata []MetadataItem
}
