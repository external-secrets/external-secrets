# Pinned release: golang:1.27.3 (Go; tracking 1.x)
FROM golang@sha256:f20d83a83e18dbd2a4714b8ffd4b30b287b6243962dd75608e86abcc63ba4efc
WORKDIR /
COPY ./bin/external-secrets /external-secrets
COPY ./bin/dlv /dlv

ENV HOME=/tmp
USER 1000:1000

EXPOSE 30000

# dlv --listen=:30000 --api-version=2 --headless=true exec /app/build/api
ENTRYPOINT ["/dlv", "--listen=:30000", "--api-version=2", "--headless=true", "--continue=true", "--accept-multiclient=true", "exec", "/external-secrets", "--"]
