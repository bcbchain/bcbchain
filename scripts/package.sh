#!/usr/bin/env bash
set -e

SDK_PACKAGE="github.com/bcbchain/sdk"
THIRD_PARTY_DIR="$(pwd)/build/third_party"
NEW_FILE="$THIRD_PARTY_DIR/src/latest_tag_new"
TAG_FILE="$THIRD_PARTY_DIR/src/latest_tag"
mkdir -p "$THIRD_PARTY_DIR/src"

downloadSDK() {
  BAK_GOPATH="$GOPATH"
  export GOPATH="$THIRD_PARTY_DIR"
  export GO111MODULE="off"
  pushd "$GOPATH" >/dev/null

  go get -d -v -u "$SDK_PACKAGE"
  go get -v -d ./...

  popd >/dev/null

  export GO111MODULE="on"
  export GOPATH="$BAK_GOPATH"
}

distSDK() {
  echo "===> disting sdk..."
  pushd "$THIRD_PARTY_DIR" >/dev/null

  tar -zcf "package.tar.gz" ./src

  popd >/dev/null
}

# Get the latest_tag from the environment, or try to figure it out.
SDK_GIT="https://api.github.com/repos/bcbchain/sdk/tags"
if [ -z "$LATEST_TAG" ]; then
  curl -s "$SDK_GIT" | grep name | head -n 1
	LATEST_TAG=$(curl -s "$SDK_GIT" | grep name | head -n 1 | awk -F "\""  '{print $4}')
fi
if [ -z "$LATEST_TAG" ]; then
    echo "Get sdk latest tag failed."
    exit 1
fi

echo "===> checking package..."
echo "$LATEST_TAG" >> "$NEW_FILE"
if [ -f "$TAG_FILE" ];then
  diff "$NEW_FILE" "$TAG_FILE" >/dev/null 2>/dev/null
  if [[ "$?" != "0" ]]; then
    echo "===> updating package..."
    rm -rf "$THIRD_PARTY_DIR/src/$SDK_PACKAGE"
    downloadSDK
  else
    echo "===> package is new..."
  fi
else
  echo "===> downloading package..."
  downloadSDK
fi
mv "$NEW_FILE" "$TAG_FILE"

distSDK

exit 0