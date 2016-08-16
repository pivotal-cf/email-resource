package email_resource_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Email Resource Suite")
}

var traceOutput = os.Getenv("TRACE") == "true"

func getPrintableCommand(command string, arguments ...string) string {
	parts := []string{"\n", "$", command}
	printableCommand := strings.Join(append(parts, arguments...), " ")
	if traceOutput {
		fmt.Println(printableCommand)
	}
	return printableCommand
}

func Run(command string, arguments ...string) string {
	output, err := RunAllowError(command, arguments...)

	if err != nil {
		fmt.Printf("\nCommand %s failed: %s", getPrintableCommand(command, arguments...), output)
		panic(err)
	}

	return output
}

func RunAllowError(command string, arguments ...string) (string, error) {
	var b bytes.Buffer
	cmd := exec.Command(command, arguments...)
	var outWriter, errWriter io.Writer = &b, &b
	if traceOutput {
		outWriter = io.MultiWriter(outWriter, os.Stdout)
		errWriter = io.MultiWriter(errWriter, os.Stderr)
	}
	cmd.Stdout = outWriter
	cmd.Stderr = errWriter
	err := cmd.Run()
	return b.String(), err
}

func RunWithStdin(stdin string, command string, arguments ...string) string {
	output, err := RunWithStdinAllowError(stdin, command, arguments...)

	if err != nil {
		fmt.Printf("\nCommand %s failed: %s", getPrintableCommand(command, arguments...), output)
		panic(err)
	}

	return output
}

func RunWithStdinAllowError(stdin string, command string, arguments ...string) (string, error) {
	var b bytes.Buffer
	cmd := exec.Command(command, arguments...)
	var outWriter, errWriter io.Writer = &b, &b
	if traceOutput {
		outWriter = io.MultiWriter(outWriter, os.Stdout)
		errWriter = io.MultiWriter(errWriter, os.Stderr)
	}
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = outWriter
	cmd.Stderr = errWriter
	err := cmd.Run()
	return b.String(), err
}
