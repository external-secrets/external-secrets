# Pinned release: alpine:3.24.3 (Alpine; tracking 3.x)
FROM alpine@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a
WORKDIR /
COPY ./bin/external-secrets /external-secrets

USER 1000:1000

ENTRYPOINT ["/external-secrets"]
