FROM gcr.io/distroless/static
ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-webhook-${TARGETOS}-${TARGETARCH} /bin/external-secrets-webhook

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets-webhook"]
