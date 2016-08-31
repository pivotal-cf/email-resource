package main

import (
	"io/ioutil"
	"os"

	"github.com/pivotal-cf/email-resource/actions/in"
)

func main() {
	inBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	in.Run(inBytes)
}