#!/usr/bin/env python3
"""Validate and render the e2e fan-out matrix defined in e2e/matrix.yaml.

Subcommands:
  check   Fail early if the matrix is inconsistent: a provider compiled into
          the suite (suites/provider/cases/import.go) is neither covered by an
          area nor declared local_only, a suite directory is not compiled in at
          all, needs_secrets disagrees with secret_groups, or an area names a
          secret group that the reusable workflow does not wire up.
  json    Print the GitHub Actions matrix (enabled areas only) as compact JSON
          for the workflow's strategy.matrix.
  plan    Print, per enabled leg, exactly which credential env vars it will
          receive. Derived from each area's secret_groups and the group -> var
          mapping parsed out of e2e-reusable.yml. This reads NO secret values
          (it never touches the secrets context), so it proves the scoping
          without any risk of leaking a value, masked or not.

Paths are resolved relative to this file, so the working directory does not
matter. YAML is read with PyYAML when present, else via yq (mikefarah), so no
new runtime dependency is required in CI.
"""

import json
import re
import subprocess
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
MATRIX = HERE / "matrix.yaml"
IMPORT = HERE / "suites/provider/cases/import.go"
WORKFLOW = HERE.parent / ".github/workflows/e2e-reusable.yml"


def load_yaml(path: Path):
    """Load a YAML file as a dict. Prefer PyYAML; fall back to yq -> JSON."""
    try:
        import yaml  # type: ignore
        return yaml.safe_load(path.read_text())
    except ModuleNotFoundError:
        out = subprocess.run(
            ["yq", "-o=json", str(path)],
            check=True, capture_output=True, text=True,
        ).stdout
        return json.loads(out)


def imported_providers() -> list[str]:
    """Provider names blank imported into the suite binary.

    Taken as the first path segment of each import, not a character class, or a
    directory like cases/gitlab-ce would resolve to "gitlab" and silently
    inherit that provider's policy."""
    return sorted({p.split("/")[0] for p in imported_paths()})


def imported_paths() -> set[str]:
    """Suite paths compiled into the suite binary, relative to cases/
    (cases/aws/secretsmanager -> aws/secretsmanager). Unlike
    imported_providers this keeps the sub-package, so it can be compared
    against the directories on disk."""
    text = IMPORT.read_text()
    return {m.group(1) for m in re.finditer(r"cases/([\w/-]+)\"", text)}


# Package-level Ginkgo container nodes. Context/When are literal aliases of
# Describe in the ginkgo DSL, and a bare It/Specify registers a spec too, so all
# of them have to count or an unimported suite using one stays invisible here.
SUITE_NODE = re.compile(
    r"^var _ = (?:F|P|X)?"
    r"(?:Describe|DescribeTable|DescribeTableSubtree|Context|When|It|Specify)\(",
    re.M,
)


def suite_dirs() -> set[str]:
    """Directories under cases/ that define a suite, relative to cases/.

    A directory is a suite when one of its own .go files registers a
    package-level Ginkgo node. That distinguishes real suites from the
    common/ helper package and from aws/, which only holds a shared
    common.go beside its three sub-suites."""
    root = IMPORT.parent
    found = set()
    for path in root.rglob("*.go"):
        if SUITE_NODE.search(path.read_text()):
            found.add(path.parent.relative_to(root).as_posix())
    return found


def group_to_vars() -> dict[str, list[str]]:
    """Map each secret group to the env vars the reusable workflow gates on it,
    parsed from lines like:
        FOO: ${{ contains(matrix.secret_groups, 'aws') && secrets.BAR || '' }}
    Reads only the workflow text, never any secret value."""
    pat = re.compile(
        r"^\s*([A-Z0-9_]+):\s*\$\{\{\s*"
        r"contains\(matrix\.secret_groups,\s*'([a-z0-9]+)'\)",
        re.MULTILINE,
    )
    mapping: dict[str, list[str]] = {}
    for var, group in pat.findall(WORKFLOW.read_text()):
        mapping.setdefault(group, []).append(var)
    for group in mapping:
        mapping[group].sort()
    return mapping


def cmd_check(matrix: dict) -> int:
    areas = matrix["areas"]
    errors: list[str] = []

    # 1. Every imported provider is either covered by an area or declared
    # local_only. local_only is for suites that need an account on an external
    # service; see the comment on the list in matrix.yaml.
    covered = {
        p for a in areas if a.get("enabled") for p in (a.get("providers") or [])
    }
    declared = {p for a in areas for p in (a.get("providers") or [])}
    local_only = set(matrix.get("local_only") or [])
    imported = imported_providers()
    missing = [p for p in imported if p not in covered and p not in local_only]
    if missing:
        errors.append(
            "providers imported into the e2e suite but neither covered by an "
            "enabled area nor listed in local_only, so they compile and run "
            "nowhere (give each a leg, or declare it local_only with a "
            "reason):\n  - " + "\n  - ".join(missing)
        )

    # 1a. local_only must stay honest: entries have to be imported, or the list
    # is stale, and must not also have a leg, or the intent is contradictory.
    stale = sorted(local_only - set(imported))
    if stale:
        errors.append(
            "local_only names providers that are not imported in import.go, so "
            "they cannot run even locally:\n  - " + "\n  - ".join(stale)
        )
    unknown = sorted(declared - set(imported))
    if unknown:
        errors.append(
            "areas name providers that are not imported in import.go, so the "
            "leg would select nothing:\n  - " + "\n  - ".join(unknown)
        )
    both = sorted(local_only & covered)
    if both:
        errors.append(
            "providers are both local_only and covered by an area, so it is "
            "unclear whether CI should run them:\n  - " + "\n  - ".join(both)
        )

    # 1b. Every suite on disk is compiled into the binary. Without this the
    # check only runs one way: a suite added under cases/ but never blank
    # imported is silently dead, which is how the akeyless, gitlab and oracle
    # suites went unrun for months while still passing this validation.
    unimported = sorted(suite_dirs() - imported_paths())
    if unimported:
        errors.append(
            "suite directories that are not blank imported in import.go, so "
            "they are never compiled into the suite binary and never run:\n  - "
            + "\n  - ".join(unimported)
        )

    # 2. needs_secrets must mirror "secret_groups is non-empty".
    for a in areas:
        has_groups = bool(a.get("secret_groups"))
        if bool(a.get("needs_secrets")) != has_groups:
            errors.append(
                f"area {a['name']!r}: needs_secrets={a.get('needs_secrets')} "
                f"disagrees with secret_groups={a.get('secret_groups')}"
            )

    # 3. Every secret group an area uses is actually wired in the workflow.
    wired = set(group_to_vars())
    for a in areas:
        for group in a.get("secret_groups") or []:
            if group not in wired:
                errors.append(
                    f"area {a['name']!r}: secret group {group!r} is not wired "
                    f"in {WORKFLOW.name} (no env var gates on it)"
                )

    if errors:
        print("ERROR: matrix.yaml is inconsistent:", file=sys.stderr)
        for e in errors:
            print(f"- {e}", file=sys.stderr)
        return 1

    enabled = sum(1 for a in areas if a.get("enabled"))
    print(
        f"matrix.yaml ok: {len(imported_providers())} providers imported, "
        f"{enabled} leg(s) enabled, {len(local_only)} local only "
        f"({', '.join(sorted(local_only)) or 'none'})"
    )
    return 0


def cmd_json(matrix: dict) -> int:
    include = [
        {
            "name": a["name"],
            "suite": a["suite"],
            "labels": a["labels"],
            "secret_groups": a.get("secret_groups") or [],
        }
        for a in matrix["areas"]
        if a.get("enabled")
    ]
    print(json.dumps({"include": include}, separators=(",", ":")))
    return 0


def cmd_plan(matrix: dict) -> int:
    """Show the credential env vars each enabled leg will receive. No secret
    values are read; the list comes from matrix.yaml + the workflow mapping."""
    mapping = group_to_vars()
    print("Per-leg credential scoping (from matrix.yaml + e2e-reusable.yml):")
    for a in matrix["areas"]:
        if not a.get("enabled"):
            continue
        groups = a.get("secret_groups") or []
        env_vars = sorted({v for g in groups for v in mapping.get(g, [])})
        shown = ", ".join(env_vars) if env_vars else "(none: in-cluster only)"
        print(f"  {a['name']}: groups={groups or '[]'} -> {shown}")
    return 0


def main() -> int:
    cmd = sys.argv[1] if len(sys.argv) > 1 else "check"
    if cmd not in ("check", "json", "plan"):
        print(f"usage: {sys.argv[0]} [check|json|plan]", file=sys.stderr)
        return 2
    matrix = load_yaml(MATRIX)
    return {"check": cmd_check, "json": cmd_json, "plan": cmd_plan}[cmd](matrix)


if __name__ == "__main__":
    sys.exit(main())
