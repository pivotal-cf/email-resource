#!/bin/bash -e

mkdir ~/.ssh/ && touch ~/.ssh/known_hosts
ssh-keyscan github.com >>~/.ssh/known_hosts

export GOPATH=$PWD/go
export PATH=$GOPATH/bin:$PATH

go get github.com/Masterminds/glide
WORKING_DIR=$GOPATH/src/github.com/pivotal-cf/email-resource
mkdir -p ${WORKING_DIR}
cp -R source/* ${WORKING_DIR}/.
cd ${WORKING_DIR}
go version
go test $(glide nv) -v
