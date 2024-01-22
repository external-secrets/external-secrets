FROM alpine@sha256:51b67269f354137895d43f3b3d810bfacd3945438e94dc5ac55fdac340352f48
WORKDIR /
COPY ./bin/external-secrets /external-secrets

ENTRYPOINT ["/external-secrets"]
