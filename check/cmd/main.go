package main

import (
	"fmt"

	"encoding/json"
	"github.com/pivotal-cf/email-resource/check"
	"os"
)

func main() {
	var input struct {
		Source struct {
			IMAP check.IMAP `json:"imap"`
		} `json:"source"`
	}

	err := json.NewDecoder(os.Stdin).Decode(&input)
	if err != nil {
		panic(err)
	}

	output, err := check.Execute(input.Source.IMAP)
	if err != nil {
		panic(err)
	}

	fmt.Println(output)
}
