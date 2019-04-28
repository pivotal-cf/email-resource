#!/bin/bash -e

mkdir ~/.ssh/ && touch ~/.ssh/known_hosts
ssh-keyscan github.com >>~/.ssh/known_hosts

export GOPATH=$PWD/go
export PATH=$GOPATH/bin:$PATH
OUTPUT_DIR=$PWD/compiled-output
SOURCE_DIR=$PWD/source

cp source/Dockerfile ${OUTPUT_DIR}/.

go get github.com/Masterminds/glide
go get github.com/xchapter7x/versioning

cd ${SOURCE_DIR}
if [ -d ".git" ]; then
  if ${DEV}; then
    ts=$(date +"%Y%m%M%S%N")
    DRAFT_VERSION="dev-${ts}"
  else
    DRAFT_VERSION=`versioning bump_patch`-`git rev-parse HEAD`
  fi
else
  DRAFT_VERSION="v0.0.0-local"
fi
echo "next version should be: ${DRAFT_VERSION}"

WORKING_DIR=$GOPATH/src/github.com/pivotal-cf/email-resource
mkdir -p ${WORKING_DIR}
cp -R ${SOURCE_DIR}/* ${WORKING_DIR}/.
cd ${WORKING_DIR}
glide install
go build -o ${OUTPUT_DIR}/bin/check ./check/cmd
go build -o ${OUTPUT_DIR}/bin/in ./in/cmd
go build -o ${OUTPUT_DIR}/bin/out -ldflags "-X main.VERSION=${DRAFT_VERSION}" ./out/cmd
echo ${DRAFT_VERSION} > ${OUTPUT_DIR}/name
echo ${DRAFT_VERSION} > ${OUTPUT_DIR}/tag
