# PROGS ?= $(notdir $(shell go list ./...))
PROGS ?= $(filter-out bin Makefile,$(shell ls))

GOOS		?= linux
BUILD_CMD	?= GOGC=off GOOS=$(GOOS) go build $(LDFLAGS) -i -o bin/$@ $@/*.go
DEPLOY_CMD	?= scp bin/$< gentoo:/tmp
RUN_CMD		?= ssh -t gentoo sudo LOGXI=${LOGXI} /tmp/$<
DEPLOY_CMD	=

ifeq ($(shell uname),Linux)
RUN_CMD		= sudo LOGXI=${LOGXI} bin/$<
else
RUN_CMD		= LOGXI=${LOGXI} bin/$<
endif

.PHONY: $(PROGS)

Q ?= @

GITSYM ?= github.com/currantlabs/$@/config.GitCommit
GITHEAD	?= $(shell git rev-parse --quiet HEAD)

BUILDDATESYM ?= github.com/currantlabs/$@/config.BuildDate
BUILDDATE	?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS		?= -ldflags="-w -X $(BUILDDATESYM)=$(BUILDDATE) -X $(GITSYM)=$(GITHEAD)"

all: $(PROGS)

$(PROGS):
	$(Q) $(BUILD_CMD)

run-%: %
	$(Q) $(DEPLOY_CMD)
	$(Q) $(RUN_CMD)
clean:
	$(Q) rm -rf bin
