root_mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
REPO_ROOT_DIR := $(realpath $(dir $(root_mkfile_path)))

export PROJECT_NAME ?= $(shell basename $(REPO_ROOT_DIR))

DIST_DIR ?= $(REPO_ROOT_DIR)/dist
SOURCES := $(wildcard $(REPO_ROOT_DIR)/cmd/*/main.go)
COMMANDS := $(patsubst $(REPO_ROOT_DIR)/cmd/%/main.go,%,$(SOURCES))
BINS := $(addprefix $(DIST_DIR)/,$(COMMANDS))

ALL_GO_FILES := $(shell find $(REPO_ROOT_DIR) -type f -name '*.go')
ALL_GO_TEST_FILES := $(shell find $(REPO_ROOT_DIR) -type f -name '*_test.go')

export GO_LD_FLAGS ?= -v -s -w -X 'main.Version=$(GIT_SHA)'

$(DIST_DIR):
	mkdir -p $(DIST_DIR)
	echo $(BINS)

$(DIST_DIR)/%: $(REPO_ROOT_DIR)/cmd/%/main.go $(DIST_DIR) $(ALL_GO_FILES)
	go build -ldflags "$(GO_LD_FLAGS)" -o $@ $(dir $<)*.go

#? build: Compile project to binary files
.PHONY: build
build: $(BINS)
