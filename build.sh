#!/bin/bash

cwd=`pwd`
export GOPATH=${cwd}:${cwd}/../bclib:${cwd}/../bcsmc-sdk:${cwd}/../../third-party

echo go install ./src/...
go install ./src/...
echo
echo go install github.com/hyperledger/burrow/cmd/bvm
go install github.com/hyperledger/burrow/cmd/bvm
echo
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install ./src/...


