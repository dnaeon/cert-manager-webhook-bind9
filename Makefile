GO ?= $(shell which go)
OS ?= $(shell $(GO) env GOOS)
ARCH ?= $(shell $(GO) env GOARCH)

IMAGE_NAME := "bind9-solver-webhook"
IMAGE_TAG := "latest"

OUT := $(shell pwd)/_out

docker:
	docker build -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

.PHONY: build
