FROM gcr.io/distroless/static@sha256:69830f29ed7545c762777507426a412f97dad3d8d32bae3e74ad3fb6160917ea
ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-${TARGETOS}-${TARGETARCH} /bin/external-secrets

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
