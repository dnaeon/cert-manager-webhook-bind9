GO ?= $(shell which go)
OS ?= $(shell $(GO) env GOOS)
ARCH ?= $(shell $(GO) env GOARCH)

IMAGE_NAME := "dnaeon/cert-manager-webhook-bind9"
IMAGE_TAG := "latest"

OUT := $(shell pwd)/_out
KUBE_VERSION=1.28.3

$(shell mkdir -p "$(OUT)")
export TEST_ASSET_ETCD=_test/kubebuilder/etcd
export TEST_ASSET_KUBE_APISERVER=_test/kubebuilder/kube-apiserver
export TEST_ASSET_KUBECTL=_test/kubebuilder/kubectl

test: _test/kubebuilder
	$(GO) test -v .

_test/kubebuilder:
	curl -fsSL https://go.kubebuilder.io/test-tools/$(KUBE_VERSION)/$(OS)/$(ARCH) -o kubebuilder-tools.tar.gz
	mkdir -p _test/kubebuilder
	tar -xvf kubebuilder-tools.tar.gz
	mv kubebuilder/bin/* _test/kubebuilder/
	rm kubebuilder-tools.tar.gz
	rm -R kubebuilder

clean: clean-kubebuilder

clean-kubebuilder:
	rm -Rf _test/kubebuilder

docker:
	docker build -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

bind9:
	docker build -t dnaeon/bind9-test-cert-manager:latest -f docker/bind9/Dockerfile docker/bind9

rendered-manifest.yaml:
	helm template \
		--set image.repository=$(IMAGE_NAME) \
		--set image.tag=$(IMAGE_TAG) \
		cert-manager-webhook-bind9 \
		deploy/cert-manager-webhook-bind9 > "$(OUT)/rendered-manifest.yaml"

helm-release:
	helm package deploy/cert-manager-webhook-bind9 -d docs/
	helm repo index ./docs --url https://dnaeon.github.io/cert-manager-webhook-bind9

.PHONY: build docker test clean clean-kubebuilder rendered-manifest.yaml helm-release
