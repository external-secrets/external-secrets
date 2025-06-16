FROM golang:1.24.4-bookworm@sha256:ee7ff13d239350cc9b962c1bf371a60f3c32ee00eaaf0d0f0489713a87e51a67 AS builder

ENV KUBECTL_VERSION="v1.28.3"
ENV HELM_VERSION="v3.13.1"

RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.6
RUN wget --max-redirect=0 -q https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl -O /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl && \
    wget --max-redirect=0 -q https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz -O - | tar -xzO linux-amd64/helm > /usr/local/bin/helm && \
    chmod +x /usr/local/bin/helm

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
WORKDIR /usr/src/app/e2e
RUN make e2e-bin

FROM alpine:3.22.0@sha256:8a1f59ffb675680d47db6337b49d22281a139e9d709335b492be023728e11715
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
