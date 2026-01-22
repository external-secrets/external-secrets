FROM golang:1.25.6-bookworm@sha256:2f768d462dbffbb0f0b3a5171009f162945b086f326e0b2a8fd5d29c3219ff14 AS builder
RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.6

FROM alpine:3.23.2@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62
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
