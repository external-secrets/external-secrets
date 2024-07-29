FROM golang:1.22.5@sha256:86a3c48a61915a8c62c0e1d7594730399caa3feb73655dfe96c7bc17710e96cf
WORKDIR /
COPY ./bin/external-secrets /external-secrets

RUN go install github.com/go-delve/delve/cmd/dlv@v1.22.0
RUN chmod +x /go/bin/dlv
RUN mv /go/bin/dlv /

EXPOSE 30000

# dlv --listen=:30000 --api-version=2 --headless=true exec /app/build/api
ENTRYPOINT ["/dlv", "--listen=:30000", "--api-version=2", "--headless=true", "--continue=true", "--accept-multiclient=true", "exec", "/external-secrets", "--"]
