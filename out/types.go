package out

import "time"

//Input - Struct that represents the input to out
type Input struct {
	Source struct {
		SMTP struct {
			Host      string
			Port      string
			Username  string
			Password  string
			Anonymous bool `json:"anonymous"`
		}
		From string
		To   []string
	}
	Params struct {
		Subject       string
		Body          string
		SendEmptyBody bool `json:"send_empty_body"`
		Headers       string
		To            string `json:"to"`
	}
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
