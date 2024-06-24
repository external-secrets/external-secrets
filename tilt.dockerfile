FROM alpine@sha256:b89d9c93e9ed3597455c90a0b88a8bbb5cb7188438f70953fede212a0c4394e0
WORKDIR /
COPY ./bin/external-secrets /external-secrets

ENTRYPOINT ["/external-secrets"]
