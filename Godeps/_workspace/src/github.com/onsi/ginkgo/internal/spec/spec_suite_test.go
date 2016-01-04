package spec_test

import (
	. "github.com/pivotal-cf/email-resource/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/pivotal-cf/email-resource/Godeps/_workspace/src/github.com/onsi/gomega"

	"testing"
)

func TestSpec(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spec Suite")
}
