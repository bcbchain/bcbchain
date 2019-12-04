#!/bin/bash

#如果编译时thirdparty出错，可能是因为thirdparty有更新，请删除thirdparty-master.zip文件重新编译
if [ ! -f "../thirdparty-master.zip" ];then
  wget https://github.com/bcbchain/thirdparty/archive/master.zip -O ../thirdparty-master.zip 
fi

if [ ! -d "../thirdparty-master" ];then
  unzip ../thirdparty-master.zip -d ..
fi

cwd=`pwd`
export GOPATH=${cwd}:${cwd}/../thirdparty-master

echo go install blockchain
go install blockchain/cmd/bcchain

echo go install tendermint
go install github.com/tendermint/tendermint/cmd/tendermint

