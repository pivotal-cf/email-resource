#!/bin/bash -e

export GOPATH=$PWD/go
export PATH=$GOPATH/bin:$PATH
OUTPUT_DIR=$PWD/compiled-output

cp source/Dockerfile ${OUTPUT_DIR}/.
cp /etc/ssl/certs/ca-certificates.crt ${OUTPUT_DIR}/ca-certificates.crt

go get github.com/tools/godep
WORKING_DIR=$GOPATH/src/github.com/pivotal-cf/email-resource
mkdir -p ${WORKING_DIR}
cp -R source/* ${WORKING_DIR}/.
cd ${WORKING_DIR}
godep go build -o ${OUTPUT_DIR}/bin/check ./actions/check
godep go build -o ${OUTPUT_DIR}/bin/in ./actions/in
godep go build -o ${OUTPUT_DIR}/bin/out ./actions/out

echo "test release name" > ${OUTPUT_DIR}/name
echo "test release tag" > ${OUTPUT_DIR}/tag
