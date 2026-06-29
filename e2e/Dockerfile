FROM golang:1.26.4-bookworm@sha256:5f68ec6805843bd3981a951ffada82a26a0bd2631045c8f7dba483fa868f5ec5 AS builder
RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.6

FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
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
