# Use distroless as minimal base image to package the exporter binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:aafe993665bf6200db38a02c11a9351e782a84b6a45e7f33b6695638c88e3931
ARG TARGETOS
ARG TARGETARCH
COPY teleport-exporter-${TARGETOS}-${TARGETARCH} /teleport-exporter
USER 65532:65532

ENTRYPOINT ["/teleport-exporter"]
