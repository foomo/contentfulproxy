.DEFAULT_GOAL:=help

TAG?=latest
#IMAGE=foomo/contentfulproxy
IMAGE=docker-registry.bestbytes.net/galeria/site/contentfulproxy
# https://hub.docker.com/repository/docker/foomo/contentfulproxy

## === Tasks ===

.PHONY: clean
## Clean build
clean:
	rm -fv bin/contentfulprox*

.PHONY: test
## Run tests
test:
	@GO_TEST_TAGS=-skip go test -coverprofile=coverage.out -race -json ./... | gotestfmt

.PHONY: lint
## Run linter
lint:
	@golangci-lint run

.PHONY: lint.fix
## Fix lint violations
lint.fix:
	@golangci-lint run --fix

.PHONY: tidy
## Run go mod tidy
tidy:
	@go mod tidy

.PHONY: outdated
## Show outdated direct dependencies
outdated:
	@go list -u -m -json all | go-mod-outdated -update -direct

## === Binary ===

.PHONY: build
## Build binary
build: clean
	go build -o bin/contentfulproxy cmd/contentfulproxy/main.go

.PHONY: build.arch
## Build arch binaries
build.arch: clean
	GOOS=linux GOARCH=amd64 go build -o bin/contentfulproxy-linux-amd64 cmd/contentfulproxy
	GOOS=darwin GOARCH=amd64 go build -o bin/contentfulproxy-darwin-amd64 cmd/contentfulproxy

## === Docker ===

.PHONY: docker.build
## Build docker image
docker.build:
	docker build -t $(IMAGE):$(TAG) .

.PHONY: docker.push
## Push docker image
docker.push:
	docker push $(IMAGE):$(TAG)

## === Utils ===

## Show help text
help:
	@awk '{ \
			if ($$0 ~ /^.PHONY: [a-zA-Z\-\_0-9]+$$/) { \
				helpCommand = substr($$0, index($$0, ":") + 2); \
				if (helpMessage) { \
					printf "\033[36m%-23s\033[0m %s\n", \
						helpCommand, helpMessage; \
					helpMessage = ""; \
				} \
			} else if ($$0 ~ /^[a-zA-Z\-\_0-9.]+:/) { \
				helpCommand = substr($$0, 0, index($$0, ":")); \
				if (helpMessage) { \
					printf "\033[36m%-23s\033[0m %s\n", \
						helpCommand, helpMessage"\n"; \
					helpMessage = ""; \
				} \
			} else if ($$0 ~ /^##/) { \
				if (helpMessage) { \
					helpMessage = helpMessage"\n                        "substr($$0, 3); \
				} else { \
					helpMessage = substr($$0, 3); \
				} \
			} else { \
				if (helpMessage) { \
					print "\n                        "helpMessage"\n" \
				} \
				helpMessage = ""; \
			} \
		}' \
		$(MAKEFILE_LIST)
