package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	var outdata struct {
		Version interface{} `json:"version"`
	}
	indata, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(indata, &outdata)
	if err != nil {
		panic(err)
	}
	if outdata.Version == nil {
		fmt.Fprintf(os.Stderr, "missing version")
		os.Exit(1)
	}
	outbytes, err := json.Marshal(outdata)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", []byte(outbytes))
}
