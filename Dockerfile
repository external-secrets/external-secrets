FROM alpine:3.13

COPY bin/external-secrets manager

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/manager"]
