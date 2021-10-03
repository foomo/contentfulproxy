SHELL := /bin/bash

TAG?=latest
IMAGE=foomo/contentfulproxy

# Utils

all: build test
tag:
	echo $(TAG)
dep:
	env GO111MODULE=on go mod download && env GO111MODULE=on go mod vendor && go install -i ./vendor/...
clean:
	rm -fv bin/contentfulprox*

# Build

build: clean
	go build -o bin/contentfulproxy
build-arch: clean
	GOOS=linux GOARCH=amd64 go build -o bin/contentfulproxy-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -o bin/contentfulproxy-darwin-amd64
build-docker: clean build-arch
	curl https://curl.haxx.se/ca/cacert.pem > .cacert.pem
	docker build -q . > .image_id
	docker tag `cat .image_id` $(IMAGE):$(TAG)
	echo "# tagged container `cat .image_id` as $(IMAGE):$(TAG)"
	rm -vf .image_id .cacert.pem

package: build
	pkg/build.sh

# Docker

docker-build:
	docker build -t $(IMAGE):$(TAG) .

docker-push:
	docker push $(IMAGE):$(TAG)

# Testing / benchmarks

test:
	go test -v ./...

bench:
	go test -run=none -bench=. ./...
