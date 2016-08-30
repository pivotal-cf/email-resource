package main

import (
	"io/ioutil"
	"os"

	"github.com/pivotal-cf/email-resource/actions/out"
)

func main() {
	sourceRoot := os.Args[1]

	inBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	out.Run(sourceRoot, inBytes)
}
