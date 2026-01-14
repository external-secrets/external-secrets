FROM gcr.io/distroless/static@sha256:cd64bec9cec257044ce3a8dd3620cf83b387920100332f2b041f19c4d2febf93

# Add metadata
LABEL maintainer="cncf-externalsecretsop-maintainers@lists.cncf.io" \
      description="External Secrets Operator is a Kubernetes operator that integrates external secret management systems"

ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-${TARGETOS}-${TARGETARCH} /bin/external-secrets

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
