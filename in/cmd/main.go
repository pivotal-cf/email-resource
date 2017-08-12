package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pivotal-cf/email-resource/in"
)

func main() {
	indata, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	output, err := in.Execute(indata)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println(output)
}
