# Makefile - build Docker image
IMAGE := pollio
TAG := dev
DOCKERFILE := Dockerfile
CONTEXT := .

FULL_IMAGE := $(IMAGE):$(TAG)

.PHONY: all build tag save load clean

all: build

build:
	docker build -f $(DOCKERFILE) -t $(FULL_IMAGE) $(CONTEXT)

tag:
	docker build -f $(DOCKERFILE) -t $(IMAGE):$(TAG) $(CONTEXT)

save:
	@test -n "$(FILE)" || (echo "set FILE=image.tar" && exit 1)
	docker save -o $(FILE) $(FULL_IMAGE)

load:
	@test -n "$(FILE)" || (echo "set FILE=image.tar" && exit 1)
	docker load -i $(FILE)

clean:
	-docker rmi $(FULL_IMAGE) || true
