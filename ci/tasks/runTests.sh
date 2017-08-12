#!/bin/bash -e

export GOPATH=$PWD/go
export PATH=$GOPATH/bin:$PATH

go get github.com/Masterminds/glide
go get golang.org/x/tools/cmd/cover
go get github.com/onsi/ginkgo/ginkgo
WORKING_DIR=$GOPATH/src/github.com/pivotal-cf/email-resource
mkdir -p ${WORKING_DIR}
cp -R source/* ${WORKING_DIR}/.
cd ${WORKING_DIR}
go test $(glide nv) -v
