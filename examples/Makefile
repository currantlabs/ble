PROGS ?= $(notdir $(shell go list ./...))

.PHONY: $(PROGS)

Q ?= @

GITSYM ?= github.com/currantlabs/$@/config.GitCommit
GITHEAD	?= $(shell git rev-parse --quiet HEAD)

BUILDDATESYM ?= github.com/currantlabs/$@/config.BuildDate
BUILDDATE	?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS		?= -ldflags="-w -X $(BUILDDATESYM)=$(BUILDDATE) -X $(GITSYM)=$(GITHEAD)"

BUILDCMD ?= go

all: prepare $(PROGS)

list:
	$(Q) echo $(PROGS)

prepare:
	$(Q) mkdir -p bin

run-%:
	sudo LOGXI=${LOGXI} go run $(patsubst run-%,%,$@)/*.go

$(PROGS):
	$(Q) echo Building bin/$@ ...
	$(Q) GOGC=off GOOS=linux $(BUILDCMD) build  $(LDFLAGS) -o bin/$@ $@/*

clean:
	$(Q) rm -rf bin
