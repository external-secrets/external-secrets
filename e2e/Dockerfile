ARG GO_VERSION=1.17
FROM golang:$GO_VERSION-buster as builder

ENV KUBECTL_VERSION="v1.21.2"
ENV HELM_VERSION="v3.7.1"

RUN go get -u github.com/onsi/ginkgo/v2/ginkgo
RUN wget -q https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl -O /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl && \
    wget -q https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz -O - | tar -xzO linux-amd64/helm > /usr/local/bin/helm && \
    chmod +x /usr/local/bin/helm

FROM alpine:3.15.0
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

COPY entrypoint.sh                  /entrypoint.sh
COPY suites/provider/provider.test  /provider.test
COPY suites/argocd/argocd.test      /argocd.test
COPY k8s                            /k8s

CMD [ "/entrypoint.sh" ]
