GO ?= $(shell which go)
OS ?= $(shell $(GO) env GOOS)
ARCH ?= $(shell $(GO) env GOARCH)

IMAGE_NAME := "dnaeon/cert-manager-webhook-bind9"
IMAGE_TAG := "latest"

OUT := $(shell pwd)/_out

docker:
	docker build -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

.PHONY: build
