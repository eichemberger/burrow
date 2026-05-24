# syntax=docker/dockerfile:1.7
#
# Build the `burrow` binary without installing Go on the host.
#
# Typical usage (binary lands in ./bin/burrow):
#
#   docker build --output type=local,dest=./bin --target export .
#
# Cross-compile for a specific OS/arch (e.g. Linux on amd64 from an arm64 Mac):
#
#   docker build \
#     --build-arg TARGETOS=linux --build-arg TARGETARCH=amd64 \
#     --output type=local,dest=./bin --target export .
#
# Override the Go toolchain version with --build-arg GO_VERSION=1.25.5

ARG GO_VERSION=1.25

FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:${GO_VERSION}-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/burrow .

# Export-only stage. Used with `docker build --output` to copy the freshly
# built binary onto the host without installing Go locally.
FROM scratch AS export
COPY --from=builder /out/burrow /burrow
