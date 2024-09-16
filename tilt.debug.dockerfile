FROM golang:1.23.1@sha256:4a3c2bcd243d3dbb7b15237eecb0792db3614900037998c2cd6a579c46888c1e
WORKDIR /
COPY ./bin/external-secrets /external-secrets

RUN go install github.com/go-delve/delve/cmd/dlv@v1.22.0
RUN chmod +x /go/bin/dlv
RUN mv /go/bin/dlv /

EXPOSE 30000

# dlv --listen=:30000 --api-version=2 --headless=true exec /app/build/api
ENTRYPOINT ["/dlv", "--listen=:30000", "--api-version=2", "--headless=true", "--continue=true", "--accept-multiclient=true", "exec", "/external-secrets", "--"]
