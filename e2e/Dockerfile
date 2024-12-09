FROM golang:1.23.4-bookworm@sha256:ef30001eeadd12890c7737c26f3be5b3a8479ccdcdc553b999c84879875a27ce as builder

ENV KUBECTL_VERSION="v1.28.3"
ENV HELM_VERSION="v3.13.1"

RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.6
RUN wget -q https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl -O /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl && \
    wget -q https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz -O - | tar -xzO linux-amd64/helm > /usr/local/bin/helm && \
    chmod +x /usr/local/bin/helm

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
WORKDIR /usr/src/app/e2e
RUN make e2e-bin

FROM alpine:3.20.3@sha256:1e42bbe2508154c9126d48c2b8a75420c3544343bf86fd041fb7527e017a4b4a
RUN apk add -U --no-cache \
    ca-certificates \
    bash \
    curl \
    tzdata \
    libc6-compat \
    openssl

COPY --from=builder /go/bin/ginkgo /usr/local/bin/
COPY --from=builder /usr/local/bin/kubectl /usr/local/bin/
COPY --from=builder /usr/local/bin/helm /usr/local/bin/

COPY --from=builder /usr/src/app/e2e/entrypoint.sh                   /entrypoint.sh
COPY --from=builder /usr/src/app/e2e/suites/provider/provider.test   /provider.test
COPY --from=builder /usr/src/app/e2e/suites/argocd/argocd.test       /argocd.test
COPY --from=builder /usr/src/app/e2e/suites/flux/flux.test           /flux.test
COPY --from=builder /usr/src/app/e2e/suites/generator/generator.test /generator.test
COPY --from=builder /usr/src/app/e2e/k8s                             /k8s

CMD [ "/entrypoint.sh" ]
