TOP := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

HUGO := $(shell which hugo)

GO := $(shell which go)

ifeq ($(GO),)
  $(error no Go found)
endif

ifeq ($(HUGO),)
  $(error hugo is not found, please install hugo (see https://gohugo.io/getting-started/installing/))
endif

ifeq ($(GOPATH),)
  $(error no GOPATH set)
endif

GOBIN := $(shell echo $${GOPATH%%:*})/bin

export PATH := $(GOBIN):$(PATH)

all: clean
	@mkdir -p $(realpath $(TOP)/..)/site
	@$(HUGO) -d $(realpath $(TOP)/..)/site --disableKinds RSS,sitemap
	@go install github.com/growler/go-imbed
	@go generate github.com/growler/go-imbed/example/...
	
clean:
	@rm -rf $(realpath $(TOP)/..)/site
	
.PHONY: all clean