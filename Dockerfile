# Use distroless as minimal base image to package the exporter binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5
ARG TARGETOS
ARG TARGETARCH
COPY teleport-exporter-${TARGETOS}-${TARGETARCH} /teleport-exporter
USER 65532:65532

ENTRYPOINT ["/teleport-exporter"]
