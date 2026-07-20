# Use distroless as minimal base image to package the exporter binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
ARG TARGETOS
ARG TARGETARCH
COPY teleport-exporter-${TARGETOS}-${TARGETARCH} /teleport-exporter
USER 65532:65532

ENTRYPOINT ["/teleport-exporter"]
