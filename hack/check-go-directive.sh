#!/usr/bin/env bash
set -euo pipefail

# check-go-directive.sh: keep the go directive of every module at a .0 patch.
#
# The go directive is the minimum Go version for anyone importing the module,
# so a patch version there forces every consumer of apis/ (and the rest) to
# bump their own go.mod for each Go point release. The Go release used to
# build ESO belongs in the toolchain directive of the root go.mod, which
# setup-go reads and which consumers ignore.

failed=0

while read -r gomod; do
    version=$(awk '$1 == "go" { print $2; exit }' "$gomod")
    if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.0$ ]]; then
        printf '%s: go directive %s must be a .0 release, e.g. go %s.0\n' \
            "$gomod" "${version:-<missing>}" "$(cut -d. -f1-2 <<< "$version")" >&2
        failed=1
    fi

    if [[ "$gomod" != "go.mod" ]] && grep -q '^toolchain ' "$gomod"; then
        printf '%s: toolchain directive belongs in the root go.mod only\n' "$gomod" >&2
        failed=1
    fi
done < <(git ls-files '*go.mod' | grep -E '(^|/)go\.mod$')

if [[ "$failed" -ne 0 ]]; then
    printf 'bump the toolchain directive in the root go.mod to build with a newer Go release\n' >&2
fi

exit "$failed"
