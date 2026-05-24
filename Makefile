.PHONY: build docker-build run test fmt vet tidy clean

BINARY := bin/burrow
VERSION ?= dev
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

# Detect host OS/arch in Go's GOOS/GOARCH naming so `make docker-build`
# produces a binary that actually runs on the host (BuildKit's TARGETOS is
# always "linux" since containers run on Linux, so we must pass these
# explicitly when we want, e.g., a darwin/arm64 binary on an Apple Silicon Mac).
HOST_OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOST_ARCH_RAW := $(shell uname -m)
HOST_ARCH := $(if $(filter x86_64,$(HOST_ARCH_RAW)),amd64,$(if $(filter aarch64 arm64,$(HOST_ARCH_RAW)),arm64,$(HOST_ARCH_RAW)))

build:
	go build $(LDFLAGS) -o $(BINARY) .

docker-build:
	DOCKER_BUILDKIT=1 docker build \
		--build-arg TARGETOS=$(HOST_OS) \
		--build-arg TARGETARCH=$(HOST_ARCH) \
		--output type=local,dest=./bin \
		--target export .

run: build
	./$(BINARY)

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/
