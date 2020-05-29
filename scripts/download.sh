#!/usr/bin/env bash

# 根据 bcb.mod 文件下载依赖资源，包括 third_party，bclib，sdk，genesis-smcrunsvc，
# 如果新加或者减少依赖，需要修改此脚本。

THIRDPARTY="third_party"
BCLIB="bclib"
SDK="sdk"
GENESIS="genesis-smcrunsvc"
CONTRACT="contract"

PREFIX="https://github.com/bcbchain/"
SUFFIX="/releases/download/"

DOWNLOAD_DIR=build/download/
rm -rf "$DOWNLOAD_DIR"
mkdir -p "$DOWNLOAD_DIR"

i=1
for _ in $(cat bcb.mod)
do
  NUM=$i
  TAG=$(awk 'NR=='$NUM' {print $1}' bcb.mod)
  VER=$(awk 'NR=='$NUM' {print $2}' bcb.mod)

  DOWNLOAD=""
  if [ "$TAG" == "$THIRDPARTY" ];then
    if [ ! -f "$THIRDPARTY""_$VER.tar.gz" ];then
      DOWNLOAD="$PREFIX$THIRDPARTY$SUFFIX$VER/$THIRDPARTY""_$VER.tar.gz"
    fi

  elif [ "$TAG" = "$BCLIB" ];then
    if [ ! -f "$BCLIB""_$VER.tar.gz" ];then
      DOWNLOAD="$PREFIX$BCLIB$SUFFIX$VER/$BCLIB""_$VER.tar.gz"
    fi

  elif [ "$TAG" = "$SDK" ];then
    if [ ! -f "$SDK""_$VER.tar.gz" ];then
      DOWNLOAD="$PREFIX$SDK$SUFFIX$VER/$SDK""_$VER.tar.gz"
    fi

  elif [ "$TAG" = "$GENESIS" ];then
    if [ ! -f "$GENESIS""_$VER.tar.gz" ];then
      DOWNLOAD="$PREFIX$CONTRACT$SUFFIX$VER/$GENESIS""_$VER.tar.gz"
    fi
  fi

  if [ -n "$DOWNLOAD" ];then
    pushd "$DOWNLOAD_DIR" || exit 1
    echo "==> downloading from" "$DOWNLOAD"
    curl -OL "$DOWNLOAD"
    popd >/dev/null  || exit 1
  fi
  : $(( i++ ))
done

echo "==> Results:"
ls -hl "$DOWNLOAD_DIR"