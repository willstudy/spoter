GODIR = $(shell go list ./... | grep -v /vendor/)
PKG := github.com/willstudy/spoter
BUILD_IMAGE ?= golang:1.9.0-alpine
GOARCH := amd64
GOOS := linux
BUILD := $(shell git rev-parse HEAD)
LDFLAGS_SPOTER := -ldflags "-X ${PKG}/cmd/mysql-operator/app.Build=${BUILD}"
