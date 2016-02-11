#!/bin/bash -exu

export TMPDIR=/tmp
export GOPATH=$PWD/go
export PATH=$GOPATH/bin:$PATH
export INPUTDIR=$PWD/go
pushd $GOPATH/src/github.com/pivotal-cf/email-resource

export GOPATH=${PWD}/Godeps/_workspace:$GOPATH
export PATH=${PWD}/Godeps/_workspace/bin:$PATH

go install github.com/onsi/ginkgo/ginkgo

ginkgo -r "$@"

go build -tags netgo -a -o bin/check ./actions/check
go build -tags netgo -a -o bin/in ./actions/in
go build -tags netgo -a -o bin/out ./actions/out

popd
cp /etc/ssl/certs/ca-certificates.crt test-and-build-docker-resource/ca-certificates.crt
cp -r $INPUTDIR/src/github.com/pivotal-cf/email-resource/bin test-and-build-docker-resource/
cp $INPUTDIR/src/github.com/pivotal-cf/email-resource/Dockerfile test-and-build-docker-resource/ 
