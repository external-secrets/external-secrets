FROM golang:1.25.4-bookworm@sha256:e17419604b6d1f9bc245694425f0ec9b1b53685c80850900a376fb10cb0f70cb AS builder
RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.6

FROM alpine:3.22.2@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412
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
