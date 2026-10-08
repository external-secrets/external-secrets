FROM alpine@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
WORKDIR /
COPY ./bin/external-secrets /external-secrets

ENTRYPOINT ["/external-secrets"]
