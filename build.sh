#!/bin/bash

if [ ! -f "../thirdparty-master.zip" ];then
  wget https://github.com/bcbchain/thirdparty/archive/master.zip -O ../thirdparty-master.zip 
fi

if [ ! -d "../thirdparty-master" ];then
  unzip ../thirdparty-master.zip -d ..
fi

cwd=`pwd`
export GOPATH=${cwd}:${cwd}/../thirdparty-master


CGO_ENABLED=0 GOOS=linux GOARCH=amd64 

echo go install blockchain
go install blockchain/cmd/bcchain

echo go install tendermint
go install github.com/tendermint/tendermint/cmd/tendermint

