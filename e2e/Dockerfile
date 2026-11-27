# Pinned release: golang:1.27.4-bookworm (Go; tracking 1.x on Debian Bookworm)
FROM golang@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS builder
RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.6

# Pinned release: alpine:3.24.3 (Alpine; tracking 3.x)
FROM alpine@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
RUN apk add -U --no-cache \
    ca-certificates \
    bash \
    curl \
    tzdata \
    libc6-compat \
    openssl

ENV KUBECTL_VERSION="v1.34.1"
ENV HELM_VERSION="v3.19.0"
RUN curl -LO https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x kubectl && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && \
    wget -q https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz -O - | tar -xzO linux-amd64/helm > /usr/local/bin/helm && \
    chmod +x /usr/local/bin/helm
COPY --from=builder /go/bin/ginkgo /usr/local/bin/

ADD e2e/entrypoint.sh /entrypoint.sh
ADD e2e/k8s /k8s
ADD e2e/suites/provider/provider.test /provider.test
ADD e2e/suites/argocd/argocd.test /argocd.test
ADD e2e/suites/flux/flux.test /flux.test
ADD e2e/suites/generator/generator.test /generator.test

CMD [ "/entrypoint.sh" ]
