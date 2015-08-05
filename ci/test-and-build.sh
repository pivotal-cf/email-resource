#!/bin/bash

set -e

export TMPDIR=/tmp
export GOPATH=$PWD/go
export PATH=$GOPATH/bin:$PATH

cd $GOPATH/src/github.com/pivotal-cf/email-resource

export GOPATH=${PWD}/Godeps/_workspace:$GOPATH
export PATH=${PWD}/Godeps/_workspace/bin:$PATH

go install github.com/onsi/ginkgo/ginkgo

ginkgo -r "$@"

go build -tags netgo -a -o bin/check ./actions/check
go build -tags netgo -a -o bin/in ./actions/in
go build -tags netgo -a -o bin/out ./actions/out

cp /etc/ssl/certs/ca-certificates.crt ca-certificates.crt
