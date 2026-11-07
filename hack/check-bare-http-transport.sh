#!/usr/bin/env bash
# Forbid constructing a bare http.Transport in production Go code.
# A zero-value transport does not use ProxyFromEnvironment, so clients
# silently bypass HTTP_PROXY / HTTPS_PROXY / NO_PROXY.
#
# Prefer: esutils.CloneDefaultHTTPTransport()
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

# Match composite literals only (not type assertions / signatures).
# Exclude tests, e2e, and hack/.
mapfile -t hits < <(rg -n --no-heading -g '*.go' -g '!*_test.go' -g '!e2e/**' -g '!hack/**' \
  -e '&http\.Transport\{' \
  -e '(^|[^[:alnum:]_])http\.Transport\{' \
  . || true)

# Drop pure-comment hits (line content before the match is only whitespace + // comment).
filtered=()
for hit in "${hits[@]+"${hits[@]}"}"; do
  file="${hit%%:*}"
  rest="${hit#*:}"
  line="${rest%%:*}"
  text="${rest#*:}"
  # Skip if the Transport literal appears only inside a // line comment.
  if [[ "${text}" =~ ^[[:space:]]*// ]]; then
    continue
  fi
  filtered+=("${hit}")
done

if ((${#filtered[@]} > 0)); then
  echo "bare http.Transport composite literals are forbidden (bypasses HTTP proxy env):" >&2
  printf '%s\n' "${filtered[@]}" >&2
  echo >&2
  echo "Use esutils.CloneDefaultHTTPTransport() instead (preserves ProxyFromEnvironment)." >&2
  echo "See runtime/esutils/http_transport.go." >&2
  exit 1
fi

echo "OK: no bare http.Transport literals in production code"
