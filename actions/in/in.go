package in

import (
	"encoding/json"
	"fmt"
	"os"
)

type Output struct {
	Version interface{} `json:"version"`
}

func Run(inBytes []byte) {
	output := Output{}
	err := json.Unmarshal(inBytes, &output)
	if err != nil {
		panic(err)
	}
	if output.Version == nil {
		fmt.Fprintf(os.Stderr, "missing version")
		os.Exit(1)
	}
	outBytes, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", []byte(outBytes))
}
