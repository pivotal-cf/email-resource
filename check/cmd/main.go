package main

import (
	"fmt"

	"github.com/pivotal-cf/email-resource/check"
)

func main() {
	output, err := check.Execute()
	if err != nil {
		panic(err)
	}
	fmt.Println(output)
}
