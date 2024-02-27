FROM gcr.io/distroless/static@sha256:072d78bc452a2998929a9579464e55067db4bf6d2c5f9cde582e33c10a415bd1
ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-${TARGETOS}-${TARGETARCH} /bin/external-secrets

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
