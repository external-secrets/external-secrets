#!/usr/bin/env bash
set -euo pipefail

provider_prefix="github.com/external-secrets/external-secrets/providers/v1"
failed=0

for provider_dir in providers/v1/*/; do
    [[ -f "${provider_dir}go.mod" ]] || continue

    if ! module_path=$(GOWORK=off go -C "$provider_dir" list -m -mod=readonly -f '{{.Path}}'); then
        failed=1
        continue
    fi

    # Load production and test imports without the workspace, which would
    # otherwise hide missing requirements by resolving sibling modules locally.
    if ! GOWORK=off go -C "$provider_dir" list -mod=readonly -deps -test ./... >/dev/null; then
        failed=1
        continue
    fi

    if ! module_graph=$(GOWORK=off go -C "$provider_dir" list -m -mod=readonly -f '{{.Path}}{{with .Replace}} => {{.Path}}{{end}}' all); then
        failed=1
        continue
    fi

    while read -r dependency arrow replacement; do
        [[ -n "${dependency:-}" ]] || continue
        [[ "$dependency" == "$module_path" ]] && continue
        [[ "$dependency" == "$provider_prefix/"* ]] || continue

        expected="../${dependency##*/}"
        if [[ "${arrow:-}" != "=>" || "${replacement:-}" != "$expected" ]]; then
            printf '%sgo.mod: provider dependency %s must have:\n' "$provider_dir" "$dependency" >&2
            printf 'replace %s => %s\n' "$dependency" "$expected" >&2
            failed=1
        fi
    done <<< "$module_graph"
done

exit "$failed"
