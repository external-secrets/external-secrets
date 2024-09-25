FROM gcr.io/distroless/static@sha256:b033683de7de51d8cce5aa4b47c1b9906786f6256017ca8b17b2551947fcf6d8
ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-${TARGETOS}-${TARGETARCH} /bin/external-secrets

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
