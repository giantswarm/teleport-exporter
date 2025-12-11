# Build the exporter binary
FROM golang:1.23.4 AS builder

WORKDIR /workspace

# Allow Go to download newer toolchain if needed
ENV GOTOOLCHAIN=auto

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY internal/ internal/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o teleport-exporter main.go

# Use distroless as minimal base image to package the exporter binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/teleport-exporter .
USER 65532:65532

ENTRYPOINT ["/teleport-exporter"]
