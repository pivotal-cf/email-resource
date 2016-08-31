#!/bin/bash
set -exu

env

go get github.com/onsi/ginkgo/ginkgo

export GOPATH=$PWD/go:$GOPATH
export INPUTDIR=$PWD/go

pushd go/src/github.com/pivotal-cf/email-resource
  ginkgo -r -p -race "$@"

  go build -tags netgo -a -o bin/check ./cmds/check
  go build -tags netgo -a -o bin/in ./cmds/in
  go build -tags netgo -a -o bin/out ./cmds/out
popd

cp /etc/ssl/certs/ca-certificates.crt test-and-build-docker-resource/ca-certificates.crt
cp -r $INPUTDIR/src/github.com/pivotal-cf/email-resource/bin test-and-build-docker-resource/
cp $INPUTDIR/src/github.com/pivotal-cf/email-resource/Dockerfile test-and-build-docker-resource/
