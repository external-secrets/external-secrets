#!/usr/bin/env bash
set -euo pipefail

# modfiles.sh: snapshot or restore the set of dirty go.mod / go.sum files.
#
# Used by the `test` Makefile target to undo the cross-submodule churn that
# `go work sync` introduces. The snapshot subcommand records which go.mod /
# go.sum files were ALREADY modified before the test run; the restore
# subcommand reverts any go.mod / go.sum files that became dirty during the
# run but were not in the snapshot, leaving the developer's intentional
# changes alone.
#
# Usage:
#   hack/modfiles.sh snapshot <snapshot-file>
#   hack/modfiles.sh restore  <snapshot-file>
#
# Both subcommands are no-ops when run outside a git checkout.

cmd="${1:-}"
snap="${2:-}"

if [[ -z "$cmd" || -z "$snap" ]]; then
    echo "Usage: $0 {snapshot|restore} <snapshot-file>" >&2
    exit 2
fi

# Print every dirty path (including renames and untracked) whose basename is
# go.mod or go.sum, one per line, sorted and unique. Tolerates non-git
# checkouts by swallowing git's error and emitting nothing.
list_dirty_modfiles() {
    { git status --porcelain 2>/dev/null || true; } \
        | awk '{print $NF}' \
        | awk '$0 ~ /(^|\/)(go\.mod|go\.sum)$/' \
        | sort -u
}

case "$cmd" in
    snapshot)
        list_dirty_modfiles > "$snap"
        ;;
    restore)
        [[ -f "$snap" ]] || exit 0
        post=$(list_dirty_modfiles)
        if [[ -n "$post" ]]; then
            to_restore=$(printf '%s\n' "$post" | grep -Fxvf "$snap" || true)
            if [[ -n "$to_restore" ]]; then
                printf '%s\n' "$to_restore" | xargs git checkout --
            fi
        fi
        rm -f "$snap"
        ;;
    *)
        echo "Unknown command: $cmd" >&2
        exit 2
        ;;
esac
