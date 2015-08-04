package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

func main() {

	var indata struct {
		Source struct {
			SMTP struct {
				Host string
			}
		}
		Params struct {
			Subject string
		}
	}

	inbytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(inbytes, &indata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "expected JSON input")
		os.Exit(1)
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
		{Name: "subject", Value: indata.Params.Subject},
	}
	outbytes, err := json.Marshal(outdata)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", []byte(outbytes))
}
