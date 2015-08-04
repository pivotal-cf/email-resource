package main

import (
	"encoding/json"
	"fmt"
	"time"
)

func main() {
	var outdata struct {
		Version struct {
			Time time.Time
		} `json:"version"`
	}
	outdata.Version.Time = time.Now().UTC()
	outbytes, err := json.Marshal(outdata)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", []byte(outbytes))
}
