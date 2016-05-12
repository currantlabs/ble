# PROGS ?= $(notdir $(shell go list ./...))
PROGS ?= $(filter-out bin Makefile,$(shell ls))

.PHONY: $(PROGS)

Q ?= @

GITSYM ?= github.com/currantlabs/$@/config.GitCommit
GITHEAD	?= $(shell git rev-parse --quiet HEAD)

BUILDDATESYM ?= github.com/currantlabs/$@/config.BuildDate
BUILDDATE	?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS		?= -ldflags="-w -X $(BUILDDATESYM)=$(BUILDDATE) -X $(GITSYM)=$(GITHEAD)"

all: $(PROGS)

list:
	$(Q) echo $(PROGS)

prepare:
	$(Q) mkdir -p bin

run-%: bin/%
	$(Q) echo Running prebuilt bin/$@ ...
	sudo LOGXI=${LOGXI} $<

$(PROGS):
	$(Q) echo Building bin/$@ ...
	$(Q) GOGC=off GOOS=linux go build  $(LDFLAGS) -o bin/$@ $@/*.go
	sudo LOGXI=${LOGXI} bin/$@

clean:
	$(Q) rm -rf bin
