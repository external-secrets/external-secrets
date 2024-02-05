FROM alpine@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b
WORKDIR /
COPY ./bin/external-secrets /external-secrets

ENTRYPOINT ["/external-secrets"]
