FROM gcr.io/distroless/static@sha256:f4a57e8ffd7ba407bdd0eb315bb54ef1f21a2100a7f032e9102e4da34fe7c196
ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-${TARGETOS}-${TARGETARCH} /bin/external-secrets

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
