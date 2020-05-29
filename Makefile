PACKAGES=$(shell go list ./...)
OUTPUT?=build/bcchain

BUILD_TAGS?=bcchain
LD_FLAGS = -X github.com/bcbchain/bcbchain/version.GitCommit=`git rev-parse --short=8 HEAD`
BUILD_FLAGS = -ldflags "$(LD_FLAGS)"
HTTPS_GIT := https://github.com/bcchain/bcchain.git
DOCKER_BUF := docker run -v $(shell pwd):/workspace --workdir /workspace bufbuild/buf
CGO_ENABLED ?= 0

# handle nostrip
ifeq (,$(findstring nostrip,$(BCBCHAIN_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
  LD_FLAGS += -s -w
endif

# allow users to pass additional flags via the conventional LDFLAGS variable
LD_FLAGS += $(LDFLAGS)

all: build
.PHONY: all

# The below include contains the tools.
 include tools.mk
# include tests.mk

###############################################################################
###                                Build BCChain                            ###
###############################################################################
build:
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_FLAGS) -tags '$(BUILD_TAGS)' -o $(OUTPUT) ./cmd/bcchain/
.PHONY: build

install:
	# Nothing to do
.PHONY: install
###############################################################################
###                              Distribution                               ###
###############################################################################

# dist builds binaries for all platforms and packages them for distribution
dist:
	@BUILD_TAGS=$(BUILD_TAGS) sh -c "'$(CURDIR)/scripts/dist.sh'"
.PHONY: dist

go-mod-cache: go.sum
	@echo "--> Download go modules to local cache"
	@go mod download
.PHONY: go-mod-cache

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify
	@go mod tidy

###############################################################################
###                            Docker image                                 ###
###############################################################################

build-docker:
	cp $(OUTPUT) DOCKER/bcbchain
	docker build --label=bcbchain --tag="bcbchain/bcbchain" DOCKER
	rm -rf DOCKER/bcbchain
.PHONY: build-docker

###############################################################################
###                            Download Contract                            ###
###############################################################################


###############################################################################
###                            Download SDK SourceCode                      ###
###############################################################################

download:
	@sh -c "'$(CURDIR)/scripts/package.sh'"
.PHONY: download_sdk

download_third_party:
	@sh -c "'$(CURDIR)/scripts/package.sh'"
.PHONY: download_sdk