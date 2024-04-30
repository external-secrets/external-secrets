#!/bin/bash
set -eu

CYAN='\033[0;36m'
NC='\033[0m'
function log() { echo -e "${CYAN}${1}${NC}"; }

function prepare_kube_config_from_host() {
  # https://github.com/microsoft/vscode-dev-containers/blob/main/containers/kubernetes-helm/.devcontainer/copy-kube-config.sh
  log '[~] Prepare .kube/config'
  if [ -d "/usr/local/share/kube-localhost" ]; then
    mkdir -p $HOME/.kube
    cp -r /usr/local/share/kube-localhost/* $HOME/.kube
    chown -R $(id -u) $HOME/.kube
    # for internal kind cluster
    sed -i -e "s/localhost/host.docker.internal/g" $HOME/.kube/config
    sed -i -e "s/127.0.0.1/host.docker.internal/g" $HOME/.kube/config
    # set insecure for remote clusters
    yq e '.clusters[].cluster."insecure-skip-tls-verify" = true' -i $HOME/.kube/config
    yq e 'del(.clusters[].cluster."certificate-authority-data")' -i $HOME/.kube/config
  fi
}

if [ -d "$HOME/.kube" ]; then
  log "[-] Kube config presents. Skip."
else
  read -p "Copy kube config from host? [y/n]" -n 1 -r
  echo
  [[ $REPLY =~ ^[Yy]$ ]] && prepare_kube_config_from_host
fi

log '[.] Done\n'