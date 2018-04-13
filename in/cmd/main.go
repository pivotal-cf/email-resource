package main

import (
	"fmt"
	"os"

	"encoding/json"
	"github.com/pivotal-cf/email-resource/check"
	"github.com/pivotal-cf/email-resource/in"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, fmt.Errorf("USAGE: %s <destination-directory>", os.Args[0]))
		os.Exit(1)
	}

	destinationDir := os.Args[1]

	var input struct {
		Source struct {
			check.IMAP `json:"imap"`
		} `json:"source"`
		Version check.Version `json:"version"`
	}

	err := json.NewDecoder(os.Stdin).Decode(&input)
	if err != nil {
		panic(err)
	}

	output, err := in.Execute(input.Source.IMAP, input.Version, destinationDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	fmt.Println(output)
}
