FROM mcr.microsoft.com/devcontainers/go:1-1.22-bookworm

USER root

# must be amd64 or arm64
ARG TARGETARCH

# BASE
RUN dnf install -y git jq make unzip && \
  dnf clean all

# YQ
RUN cd /tmp && \
  version=4.35.2 && \
  curl -L --output /usr/local/bin/yq https://github.com/mikefarah/yq/releases/download/v${version}/yq_linux_${TARGETARCH} && \
  chmod +x /usr/local/bin/yq

# DOCKER-CLI
RUN cd /tmp && \
  version=24.0.6 && \
  arch=${TARGETARCH} && \
  [ "$arch" = "arm64" ] && arch="aarch64"; \
  [ "$arch" = "amd64" ] && arch="x86_64"; \
  curl -L https://download.docker.com/linux/static/stable/${arch}/docker-${version}.tgz | tar xz && \
  mv docker/docker /usr/local/bin/

# KUBECTL
RUN cd /tmp && \
  curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/${TARGETARCH}/kubectl" && \
  chmod +x kubectl && mv kubectl /usr/local/bin/

## KUBELOGIN
RUN cd /tmp && \
    version=0.0.33 && \
    curl -L https://github.com/Azure/kubelogin/releases/download/v${version}/kubelogin-linux-${TARGETARCH}.zip > ./kubelogin-linux-${TARGETARCH}.zip && \
    unzip /tmp/kubelogin-linux-${TARGETARCH}.zip && \
    mv /tmp/bin/linux_${TARGETARCH}/kubelogin /usr/local/bin

# HELM
RUN cd /tmp && \
  version=3.14.0 && \
  curl -L https://get.helm.sh/helm-v${version}-linux-${TARGETARCH}.tar.gz | tar xz && \
  mv linux-${TARGETARCH}/helm /usr/local/bin/helm

# KIND
RUN cd /tmp && \
  version=0.20.0 && \
  curl -Lo ./kind https://kind.sigs.k8s.io/dl/v${version}/kind-linux-${TARGETARCH} && \
  chmod +x ./kind && mv ./kind /usr/local/bin/

# KREW
RUN cd /tmp && \
  version=0.4.4 && \
  curl -L https://github.com/kubernetes-sigs/krew/releases/download/v${version}/krew-linux_${TARGETARCH}.tar.gz | tar xz && \
  mv krew-linux_${TARGETARCH} /usr/local/bin/kubectl-krew && \
  echo 'export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"' >> $HOME/.bashrc

# KREW PLUGINS
RUN kubectl krew install ctx ns

# K9S
RUN cd /tmp && \
  version=0.27.4 && \
  curl -L https://github.com/derailed/k9s/releases/download/v${version}/k9s_Linux_${TARGETARCH}.tar.gz | tar xz && \
  mv k9s /usr/local/bin/
