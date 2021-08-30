FROM alpine:3.14.1
ARG TARGETOS
ARG TARGETARCH
COPY bin/external-secrets-${TARGETOS}-amd64 /bin/external-secrets 
#Change back to Targetarch

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
