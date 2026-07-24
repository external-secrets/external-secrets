#!/usr/bin/env python3
"""Validate and render the e2e fan-out matrix defined in e2e/matrix.yaml.

Subcommands:
  check   Fail early if the matrix is inconsistent: a provider compiled into
          the suite (suites/provider/cases/import.go) is not covered by any
          area, needs_secrets disagrees with secret_groups, or an area names a
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
    """Provider names compiled into the suite: the segment after cases/ in
    each blank import of import.go (cases/aws/secretsmanager -> aws)."""
    text = IMPORT.read_text()
    return sorted({m.group(1) for m in re.finditer(r"cases/([a-z0-9]+)", text)})


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

    # 1. Every imported provider is covered by some area.
    covered = {p for a in areas for p in (a.get("providers") or [])}
    missing = [p for p in imported_providers() if p not in covered]
    if missing:
        errors.append(
            "providers imported into the e2e suite but not covered by any "
            "area (add each to an area's providers list and a leg):\n  - "
            + "\n  - ".join(missing)
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
        f"matrix.yaml ok: {len(imported_providers())} providers covered, "
        f"{enabled} leg(s) enabled"
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
