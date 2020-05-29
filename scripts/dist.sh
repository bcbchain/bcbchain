#!/usr/bin/env bash
set -e

# WARN: non hermetic build (people must run this script inside docker to
# produce deterministic binaries).

bash scripts/download.sh

# Get the version from the environment, or try to figure it out.
if [ -z "$VERSION" ]; then
	VERSION=$(awk -F\" '/BCBChainSemVer =/ { print $2; exit }' < version/version.go)
fi
if [ -z "$VERSION" ]; then
    echo "Please specify a version."
    exit 1
fi

VERSION="v$VERSION"
echo "==> Building version $VERSION..."

# Delete the old dir
echo "==> Removing old directory..."
rm -rf build/pkg
mkdir -p build/pkg

# Get the git commit
GIT_COMMIT="$(git rev-parse --short=8 HEAD)"
GIT_IMPORT="github.com/bcbchain/bcbchain/version"

# Determine the arch/os combos we're building for
XC_ARCH=${XC_ARCH:-"amd64"}   # 386 arm
XC_OS=${XC_OS:-"solaris darwin freebsd linux windows"}
XC_EXCLUDE=${XC_EXCLUDE:-" darwin/arm solaris/amd64 solaris/386 solaris/arm freebsd/amd64 windows/arm "}

# Make sure build tools are available.
#make tools
# Make sure download contract

#=====
DOWNLOAD_DIR=build/download/
pushd "$DOWNLOAD_DIR" >/dev/null

BCLIBNAME=""
SDKNAME=""
THIRDPARTYNAME=""
for FILENAME in $(find . -mindepth 1 -maxdepth 1 -type f); do
  if [[ "$FILENAME" == *bclib*.tar.gz ]]; then
    BCLIBNAME=${FILENAME:2}
  fi

  if [[ "$FILENAME" == *sdk*.tar.gz ]]; then
    SDKNAME=${FILENAME:2}
  fi

  if [[ "$FILENAME" == *third_party*.tar.gz ]]; then
    THIRDPARTYNAME=${FILENAME:2}
  fi
done

mkdir -p bclib
if [ -n "$BCLIBNAME" ];then
  echo "$BCLIBNAME"
  tar xf "$BCLIBNAME" -C bclib/
fi

mkdir -p sdk
if [ -n "$SDKNAME" ];then
  tar xf "$SDKNAME" -C sdk/
fi

mkdir -p src
tar xf third_party*.tar.gz -C src/

mkdir -p src/github.com/bcbchain/
cp -r bclib/* src/github.com/bcbchain/
cp -r sdk/* src/github.com/bcbchain/

if [ -n "$THIRDPARTYNAME" ];then
  tar zcf "$THIRDPARTYNAME" src
fi

rm -rf src
mkdir -p src/blockchain/smcsdk
mkdir -p src/common
mkdir -p src/github.com

cp -r bclib/algorithm src/blockchain/
rm -rf bclib/algorithm

cp -r bclib/types src/common/
rm -rf bclib/types

cp -r bclib/tendermint src/github.com/
rm -rf bclib/tendermint

cp -r bclib/* src/common/
rm -rf bclib

cp -r sdk/* src/blockchain/
tar zcf "$SDKNAME" src
rm -rf src
rm -rf sdk

popd >/dev/null

# copy bundle direction
echo "==> copy bundle direction..."
IFS=' ' read -ra arch_list <<< "$XC_ARCH"
IFS=' ' read -ra os_list <<< "$XC_OS"
for arch in "${arch_list[@]}"; do
	for os in "${os_list[@]}"; do
		if [[ "$XC_EXCLUDE" !=  *" $os/$arch "* ]]; then
			echo "--> copy to $os/$arch"
      cp -rf bundle/setup "build/pkg/${os}_${arch}/"
      if [ -f "build/pkg/${os}_${arch}/pieces/smcrunsvc_${os}" ];then
        mv "build/pkg/${os}_${arch}/pieces/smcrunsvc_${os}" "build/pkg/${os}_${arch}/pieces/smcrunsvc"
      else
        mv "build/pkg/${os}_${arch}/pieces/smcrunsvc.exe_${os}" "build/pkg/${os}_${arch}/pieces/smcrunsvc.exe"
      fi
      rm -f "build/pkg/${os}_${arch}/pieces"/smcrunsvc*_*
      for CHAINID in $(find ./bundle/.config -mindepth 1 -maxdepth 1 -type d); do
	      CHAIN=$(basename "${CHAINID}")
	      echo "--> ${CHAIN}"

        mkdir -p "build/pkg/${os}_${arch}/pieces/$CHAIN/.config"
	      cp -f "bundle/.config/$CHAIN"/* "build/pkg/${os}_${arch}/pieces/$CHAIN/.config/"
	      cp build/download/genesis* "build/pkg/${os}_${arch}/pieces/$CHAIN/.config/"
	      cp build/download/third_party*.tar.gz "build/pkg/${os}_${arch}/pieces/"
	      cp build/download/sdk*.tar.gz "build/pkg/${os}_${arch}/pieces/"
      done
		fi
	done
done
echo

# Build!
# ldflags: -s Omit the symbol table and debug information.
#	         -w Omit the DWARF symbol table.
echo "==> Building..."
IFS=' ' read -ra arch_list <<< "$XC_ARCH"
IFS=' ' read -ra os_list <<< "$XC_OS"
for arch in "${arch_list[@]}"; do
	for os in "${os_list[@]}"; do
		if [[ "$XC_EXCLUDE" !=  *" $os/$arch "* ]]; then
			echo "--> $os/$arch"
			GOOS=${os} GOARCH=${arch} go build -ldflags "-s -w -X ${GIT_IMPORT}.GitCommit=${GIT_COMMIT}" -tags="${BUILD_TAGS}" -o "build/pkg/${os}_${arch}/pieces/bcchain" ./cmd/bcchain
		fi
	done
done

# tar compress all the files.
echo "==> Packaging..."
for PLATFORM in $(find ./build/pkg -mindepth 1 -maxdepth 1 -type d); do
	OSARCH=$(basename "${PLATFORM}")
	echo "--> ${OSARCH}"

	pushd "$PLATFORM" >/dev/null 2>&1
	tar -zcf "../${OSARCH}.tar.gz" ./*
	popd >/dev/null 2>&1
done

# Add "bcchain" and $VERSION prefix to package name.
rm -rf ./build/dist
mkdir -p ./build/dist
for FILENAME in $(find ./build/pkg -mindepth 1 -maxdepth 1 -type f); do
  FILENAME=$(basename "$FILENAME")
	cp "./build/pkg/${FILENAME}" "./build/dist/bcbchain_${VERSION}_${FILENAME}"
done

# Make the checksums.
pushd ./build/dist
shasum -a256 ./* > "./bcbchain_${VERSION}_SHA256SUMS"
popd

# Done
echo
echo "==> Results:"
ls -hl ./build/dist

exit 0