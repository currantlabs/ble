# PROGS ?= $(notdir $(shell go list ./...))
PROGS ?= $(filter-out bin Makefile devices,$(shell ls))

BUILD_CMD	?= GOGC=off GOARCH=$(GOARCH) GOOS=$(GOOS) go build $(LDFLAGS) -i -o bin/$@ $@/*.go
ARGS		?=

ifeq ($(TARGET),gentoo)
GOOS		?= linux
DEPLOY_CMD	?= scp bin/$< gentoo:/tmp
RUN_CMD		?= ssh -t gentoo sudo LOGXI=${LOGXI} /tmp/$< ${ARGS}
endif

ifeq ($(TARGET),herc)
GOOS		?= linux
GOARCH		?= arm
DEPLOY_CMD	?= scp bin/$< herc:/tmp
RUN_CMD		?= ssh -t herc LOGXI=${LOGXI} /tmp/$< ${ARGS}
endif

ifeq ($(shell uname),Linux)
GOOS		?= linux
RUN_CMD		?= sudo LOGXI=${LOGXI} bin/$< ${ARGS}
else
GOOS		?= darwin
RUN_CMD		?= LOGXI=${LOGXI} bin/$< ${ARGS}
endif

.PHONY: $(PROGS)

Q ?= @

GITSYM		?= github.com/currantlabs/$@/config.GitCommit
GITHEAD		?= $(shell git rev-parse --quiet HEAD)
BUILDDATESYM	?= github.com/currantlabs/$@/config.BuildDate
BUILDDATE	?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS		?= -ldflags="-w -X $(BUILDDATESYM)=$(BUILDDATE) -X $(GITSYM)=$(GITHEAD)"

all: $(PROGS)

build:
	GOOS=linux make -j
	GOOS=darwin make -j

$(PROGS):
	$(Q) $(BUILD_CMD)

run-%: %
	$(Q) $(DEPLOY_CMD)
	$(Q) $(RUN_CMD)
clean:
	$(Q) rm -rf bin
