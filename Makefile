GODIR = $(shell go list ./... | grep -v /vendor/)
PKG := github.com/willstudy/spoter
BUILD_IMAGE ?= golang:1.9.0-alpine
GOARCH := amd64
GOOS := linux
BUILD := $(shell git rev-parse HEAD)
LDFLAGS_SPOTER := -ldflags "-X ${PKG}/cmd/spoter/app.Build=${BUILD}"

all: image-spoter

clean:
	@echo "clean <release go .go> dirs"
	@rm -rf release .go go
.PHONY: pre-build

build-dirs:
	@mkdir -p .go/src/$(PKG) ./go/bin
	@mkdir -p release
.PHONY: build-dirs

build-spoter-controller: build-dirs
	@docker run                                                            \
	    --rm                                                               \
	    -ti                                                                \
	    -u $$(id -u):$$(id -g)                                             \
	    -v $$(pwd)/.go:/go                                                 \
	    -v $$(pwd):/go/src/$(PKG)                                          \
	    -v $$(pwd)/release:/go/bin                                         \
	    -e GOOS=$(GOOS)                                                    \
	    -e GOARCH=$(GOARCH)                                                \
	    -e CGO_ENABLED=0                                                   \
	    -w /go/src/$(PKG)                                                  \
	    $(BUILD_IMAGE)                                                     \
	    go install -v -pkgdir /go/pkg $(LDFLAGS_SPOTER) ./cmd/spoter
.PHONY: build-spoter-controller

image-spoter: build-spoter-controller
	@sh build/spoter.sh ${registry}
.PHONY: image-spoter
