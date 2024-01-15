FROM golang:1.21.6
WORKDIR /
COPY ./bin/external-secrets /external-secrets

RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN chmod +x /go/bin/dlv
RUN mv /go/bin/dlv /

EXPOSE 30000

# dlv --listen=:30000 --api-version=2 --headless=true exec /app/build/api
ENTRYPOINT ["/dlv", "--listen=:30000", "--api-version=2", "--headless=true", "--continue=true", "--accept-multiclient=true", "exec", "/external-secrets", "--"]
