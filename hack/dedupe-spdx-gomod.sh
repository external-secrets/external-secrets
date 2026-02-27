#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  dedupe-spdx-gomod.sh --input INPUT_SPDX_JSON --output OUTPUT_SPDX_JSON [--drop-file-ownership]

Description:
  Deduplicates SPDX package nodes by purl (fallback: name@version), rewrites
  relationships to canonical SPDX IDs, and deduplicates relationships.

  Optional flag --drop-file-ownership removes file ownership-heavy data:
  - drops relationshipType OTHER
  - removes files[] entries
EOF
}

INPUT=""
OUTPUT=""
DROP_FILE_OWNERSHIP="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --input)
      if [[ -z "${2:-}" || "${2:-}" == --* || "${2:-}" == -* ]]; then
        echo "Error: --input requires a value" >&2
        usage >&2
        exit 1
      fi
      INPUT="${2:-}"
      shift 2
      ;;
    --output)
      if [[ -z "${2:-}" || "${2:-}" == --* || "${2:-}" == -* ]]; then
        echo "Error: --output requires a value" >&2
        usage >&2
        exit 1
      fi
      OUTPUT="${2:-}"
      shift 2
      ;;
    --drop-file-ownership)
      DROP_FILE_OWNERSHIP="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "${INPUT}" || -z "${OUTPUT}" ]]; then
  usage >&2
  exit 1
fi

if [[ ! -f "${INPUT}" ]]; then
  echo "Input file not found: ${INPUT}" >&2
  exit 1
fi

TMP_OUT="$(mktemp)"

jq \
  --argjson drop_file_ownership "${DROP_FILE_OWNERSHIP}" '
  def purl:
    ((.externalRefs // [])
      | map(select(.referenceType == "purl") | .referenceLocator)
      | first);
  def package_key:
    if (purl // "") != "" then
      "purl|" + purl
    else
      # No purl means identity is uncertain across ecosystems/catalogers.
      # Keep the key non-destructive by including provenance fields and SPDXID.
      "nopurl|spdxid=" + (.SPDXID // "") +
      "|name=" + (.name // "") +
      "|version=" + (.versionInfo // "") +
      "|supplier=" + (.supplier // "") +
      "|sourceInfo=" + (.sourceInfo // "")
    end;

  . as $doc
  | ($doc.packages // [] | map(. + {__dedupe_key: package_key})) as $pkgs
  | ($pkgs
      | sort_by(.__dedupe_key)
      | group_by(.__dedupe_key)
      | map({
          canonical_spdxid: .[0].SPDXID,
          all_spdxids: map(.SPDXID),
          canonical_pkg: (.[0] | del(.__dedupe_key))
        })) as $groups
  | ($groups | map(.canonical_pkg)) as $new_packages
  | ($groups | map(.all_spdxids[] as $old | {($old): .canonical_spdxid}) | add // {}) as $id_map
  | ($doc.relationships // []
      | map(
          .spdxElementId = ($id_map[.spdxElementId] // .spdxElementId)
          | .relatedSpdxElement = ($id_map[.relatedSpdxElement] // .relatedSpdxElement)
        )
      | if $drop_file_ownership then
          map(select(.relationshipType != "OTHER"))
        else
          .
        end
      | unique_by(.spdxElementId + "|" + .relationshipType + "|" + .relatedSpdxElement)
    ) as $new_relationships
  | $doc
  | .packages = $new_packages
  | .relationships = $new_relationships
  | .documentDescribes = (($doc.documentDescribes // []) | map($id_map[.] // .) | unique)
  | if $drop_file_ownership then del(.files) else . end
' "${INPUT}" > "${TMP_OUT}"

mv "${TMP_OUT}" "${OUTPUT}"
