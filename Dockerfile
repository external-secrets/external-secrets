FROM gcr.io/distroless/static@sha256:d9f9472a8f4541368192d714a995eb1a99bab1f7071fc8bde261d7eda3b667d8

# Add metadata
LABEL maintainer="cncf-externalsecretsop-maintainers@lists.cncf.io" \
      description="External Secrets Operator is a Kubernetes operator that integrates external secret management systems"

ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-${TARGETOS}-${TARGETARCH} /bin/external-secrets

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
