DOCKER := docker
BUILD_DATE:=$(shell date --rfc-3339=seconds)
GIT_SHA:=$(shell git rev-parse HEAD)

REGISTRY?=docker.io/stevenfraser
BASE_OS=ubuntu
BASE_TAG=20.04

IMAGE_NAME?=$(REGISTRY)/byoh-bundle-$(BASE_OS)_$(BASE_TAG).1_x86-64_k8s
CONTAINERD_VERSION?=1.6.20
TAG?=$(shell git rev-parse --short HEAD)

RELEASE_VERSIONS?=1.26.3 # RELEASE follows the kubernetes release

build: $(addprefix build-,$(RELEASE_VERSIONS))
build-%:
	$(DOCKER) build -t $(IMAGE_NAME):v$*  \
		--build-arg KUBERNETES_VERSION=$* \
		--build-arg CONTAINERD_VERSION=$(CONTAINERD_VERSION) \
		--build-arg BASE_OS=$(BASE_OS) \
		--build-arg BASE_OS_VERSION=$(BASE_TAG) \
		--label "org.opencontainers.image.created"="$(BUILD_DATE)" \
		--label "org.opencontainers.image.revision"="$(GIT_SHA)" \
		.

push: $(addprefix push-,$(RELEASE_VERSIONS))
push-%:
	$(DOCKER) push $(IMAGE_NAME):v$*

#registry.hub.docker.com/stevenfraser/byoh-bundle-ubuntu_20.04.1_x86-64_k8s:v1.26.3