# Pinned release: gcr.io/distroless/static-debian13:latest (Distroless static; Debian 13)
FROM gcr.io/distroless/static@sha256:1a64f46364620465ef9e9a8f5413757dfdf22695e050bbfd7303eeaf17a59383

# Add metadata
LABEL maintainer="cncf-externalsecretsop-maintainers@lists.cncf.io" \
      description="External Secrets Operator is a Kubernetes operator that integrates external secret management systems"

ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-${TARGETOS}-${TARGETARCH} /bin/external-secrets

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
