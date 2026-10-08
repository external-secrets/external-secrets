FROM gcr.io/distroless/static@sha256:58133991db06659feaabe0f4e97a35cebf15ef4ea08f8a4c6d2ee5f75e4aa6a0

# Add metadata
LABEL maintainer="cncf-externalsecretsop-maintainers@lists.cncf.io" \
      description="External Secrets Operator is a Kubernetes operator that integrates external secret management systems"

ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-${TARGETOS}-${TARGETARCH} /bin/external-secrets

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
