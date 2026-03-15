FROM gcr.io/distroless/static@sha256:28efbe90d0b2f2a3ee465cc5b44f3f2cf5533514cf4d51447a977a5dc8e526d0

# Add metadata
LABEL maintainer="cncf-externalsecretsop-maintainers@lists.cncf.io" \
      description="External Secrets Operator is a Kubernetes operator that integrates external secret management systems"

ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-${TARGETOS}-${TARGETARCH} /bin/external-secrets

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
