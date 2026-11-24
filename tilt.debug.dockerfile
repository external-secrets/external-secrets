# Pinned release: golang:1.27.4 (Go; tracking 1.x)
FROM golang@sha256:205c7636d74e0ffbe3d007162df7d74f859f5a8c8322dfb9f907977686c126cf
WORKDIR /
COPY ./bin/external-secrets /external-secrets
COPY ./bin/dlv /dlv

ENV HOME=/tmp
USER 1000:1000

EXPOSE 30000

# dlv --listen=:30000 --api-version=2 --headless=true exec /app/build/api
ENTRYPOINT ["/dlv", "--listen=:30000", "--api-version=2", "--headless=true", "--continue=true", "--accept-multiclient=true", "exec", "/external-secrets", "--"]
