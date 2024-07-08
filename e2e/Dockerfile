FROM golang:1.22.5-bookworm@sha256:6c2780255bb7b881e904e303be0d7a079054160b2ce1efde446693c0850a39ad as builder

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

FROM alpine:3.20.1@sha256:b89d9c93e9ed3597455c90a0b88a8bbb5cb7188438f70953fede212a0c4394e0
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
