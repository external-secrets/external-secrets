FROM gcr.io/distroless/static
ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-${TARGETOS}-${TARGETARCH} /bin/external-secrets
COPY bin/external-secrets-${TARGETOS}-${TARGETARCH}-providerless /bin/external-secrets-providerless
COPY bin/providers/${TARGETARCH} /bin

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
