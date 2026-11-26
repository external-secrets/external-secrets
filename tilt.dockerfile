# Pinned release: alpine:3.24.3 (Alpine; tracking 3.x)
FROM alpine@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
WORKDIR /
COPY ./bin/external-secrets /external-secrets

USER 1000:1000

ENTRYPOINT ["/external-secrets"]
