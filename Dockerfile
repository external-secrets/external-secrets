FROM alpine:3.14.0

COPY bin/external-secrets /bin/external-secrets

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
