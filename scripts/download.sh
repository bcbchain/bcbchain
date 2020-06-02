#!/usr/bin/env bash

# 根据 bcb.mod 文件下载依赖资源，包括 third_party，bclib，sdk，genesis-smcrunsvc，

echo "==> Downloading files..."
PREFIX="https://"
SUFFIX="/releases/download/"

DOWNLOAD_DIR=build/download/
rm -rf "$DOWNLOAD_DIR"
mkdir -p "$DOWNLOAD_DIR"

i=1
for _ in $(cat scripts/bcb.mod)
do
  NUM=$i
  TAG=$(awk 'NR=='$NUM' {print $1}' scripts/bcb.mod)
  VER=$(awk 'NR=='$NUM' {print $2}' scripts/bcb.mod)

  if [[ "$TAG" == "" ]];then
    continue
  fi

  if [[ "$VER" == "go.mod" ]];then
    ii=1
    for _ in $(cat go.mod)
    do
      N=$ii
      GTAG=$(awk 'NR=='$N' {print $1}' go.mod)
      GVER=$(awk 'NR=='$N' {print $2}' go.mod)

      if [[ "$GTAG" == "$TAG" ]];then
        VER="$GVER"
        break
      fi
      : $(( ii++ ))
    done
  fi

  FILENAME="${TAG##*/}"
  DOWNLOAD="$PREFIX$TAG$SUFFIX$VER/$FILENAME""_$VER.tar.gz"

  pushd "$DOWNLOAD_DIR" >/dev/null || exit 1
  echo "==> downloading from" "$DOWNLOAD"
  curl -OL "$DOWNLOAD"
  popd >/dev/null  || exit 1
  : $(( i++ ))
done

echo "==> Download results:"
ls -hl "$DOWNLOAD_DIR"