all: mailslurper

BUILD_ID := $(if $(BUILD_ID),$(BUILD_ID),$(shell git rev-parse --short HEAD))

BUILD_DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS=-ldflags "-X main.BuildId=$(BUILD_ID) -X main.BuildDate=$(BUILD_DATE)"
ARCH := $(shell uname -m)
OS_NAME := $(shell uname -s | tr A-Z a-z)

ifeq ($(ARCH),aarch64)
	ARCH=arm64
endif

ifeq ($(ARCH),x86_64)
	ARCH=amd64
endif

ifeq ($(DEBUG),true)
	GCFLAGS=-gcflags="all=-N -l"
endif
GCFLAGS=-gcflags="all=-N -l"

prebuild:
	mkdir -p bin/arm64/linux bin/amd64/linux bin/amd64/windows bin/amd64/darwin

mailslurper: prebuild
	GOOS=$(OS_NAME) GOARCH=$(ARCH) go build $(GCFLAGS) $(LDFLAGS) -o bin/$(ARCH)/$(OS_NAME)/mailslurper mailslurper/*.go 
