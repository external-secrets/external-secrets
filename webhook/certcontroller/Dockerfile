FROM gcr.io/distroless/static
ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-cert-controller-${TARGETOS}-${TARGETARCH} /bin/external-secrets-cert-controller

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets-cert-controller"]
