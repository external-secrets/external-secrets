#!/usr/bin/env python3
"""Validate and render the e2e fan-out matrix defined in e2e/matrix.yaml.

Subcommands:
  check   Fail early if the matrix is inconsistent: a provider compiled into
          the suite (suites/provider/cases/import.go) is not covered by any
          area, needs_secrets disagrees with secret_groups, or an area names a
          secret group that the reusable workflow does not wire up.
  json    Print the GitHub Actions matrix as compact JSON for the workflow's
          strategy.matrix. With --changed <file|->, keep only the legs that
          change can affect. See "Affected-only selection" in e2e/README.md.
  selftest Check the affected-only resolver against a table of known cases.
  plan    Print, per enabled leg, exactly which credential env vars it will
          receive. Derived from each area's secret_groups and the group -> var
          mapping parsed out of e2e-reusable.yml. This reads NO secret values
          (it never touches the secrets context), so it proves the scoping
          without any risk of leaking a value, masked or not.

Paths are resolved relative to this file, so the working directory does not
matter. YAML is read with PyYAML when present, else via yq (mikefarah), so no
new runtime dependency is required in CI.
"""

import io
import json
import re
import subprocess
import sys
from contextlib import redirect_stderr
from fnmatch import fnmatchcase
from pathlib import Path

USAGE = "usage: matrix.py [check|json [--changed <file|->]|plan|selftest]"
Match = tuple[str, str] | None

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


def tracked_files() -> list[str]:
    """Repo-relative tracked paths. The glob rules in check need to know that a
    pattern corresponds to something real."""
    out = subprocess.run(
        ["git", "-C", str(HERE.parent), "ls-files"],
        check=True, capture_output=True, text=True,
    ).stdout
    return out.splitlines()


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

    # 4. Selection depends on paths, so an enabled area without them would run
    # only when full_matrix_paths hits and silently sit out every other PR.
    for a in areas:
        if a.get("enabled") and not a.get("paths"):
            errors.append(
                f"area {a['name']!r}: enabled but declares no paths, so "
                "affected-only selection would almost never run it"
            )

    shared = matrix.get("full_matrix_paths") or []
    live = [a for a in areas if a.get("enabled")]
    files = tracked_files()

    # 5. Every glob must match something. A typo like providers/v1/Azure/**
    # stops selecting its leg while check and selftest stay green, and unlike
    # enabled: false it leaves no trace on later pull requests.
    labelled = [("full_matrix_paths", g) for g in shared]
    labelled += [(f"area {a['name']!r}", g) for a in live
                 for g in (a.get("paths") or [])]
    for where, glob in labelled:
        if not any(fnmatchcase(f, glob) for f in files):
            errors.append(f"{where}: glob {glob!r} matches no tracked file")

    # 6. Every suite's own files must reach some leg. They sit one level above
    # the per-case globs, so the provider suite's bootstrap was selected by
    # nothing until it was listed explicitly.
    selectable = shared + [g for a in live for g in (a.get("paths") or [])]
    for f in files:
        parts = f.split("/")
        if len(parts) == 4 and parts[:2] == ["e2e", "suites"]:
            if not any(fnmatchcase(f, g) for g in selectable):
                errors.append(
                    f"suite file {f} is selected by no enabled leg, so a "
                    "change to it would skip the legs that run it"
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


def parse_changed(text: str, source: str) -> list[str] | None:
    """Non-blank lines of text, or None for "nothing usable, run everything".
    An empty list is indistinguishable from a diff step that produced
    nothing, so it must not narrow the matrix."""
    paths = [line.strip() for line in text.splitlines() if line.strip()]
    if not paths:
        print(f"WARNING: no changed paths in {source}; running the full matrix",
              file=sys.stderr)
        return None
    return paths


def read_changed(source: str) -> list[str] | None:
    """Changed paths, one per line, from a file or stdin ("-"). None means
    "run everything": an unreadable file is a broken diff step, not an empty
    diff, so it may not narrow the matrix either."""
    try:
        text = sys.stdin.read() if source == "-" else Path(source).read_text()
    except OSError as err:
        print(f"WARNING: cannot read changed paths from {source!r} ({err}); "
              "running the full matrix", file=sys.stderr)
        return None
    return parse_changed(text, repr(source))


def first_match(patterns: list[str], paths: list[str]) -> Match:
    """First (pattern, path) pair that matches, else None. fnmatchcase, not
    fnmatch: the latter normalises case per platform, so a laptop and a Linux
    runner would disagree."""
    for pattern in patterns:
        for path in paths:
            if fnmatchcase(path, pattern):
                return pattern, path
    return None


def select_areas(matrix: dict, changed: list[str] | None) -> list[dict]:
    """Enabled areas a change can affect; changed=None runs all of them. An
    area is kept when it is always-on or one of its paths globs matches, and
    full_matrix_paths keeps every area. See matrix.yaml for why that is not
    merely defensive."""
    enabled = [a for a in matrix["areas"] if a.get("enabled")]
    if changed is None:
        return enabled

    if hit := first_match(matrix.get("full_matrix_paths") or [], changed):
        print(f"full matrix: {hit[1]} matches full_matrix_paths {hit[0]!r}",
              file=sys.stderr)
        return enabled

    selected, dropped = [], []
    for a in enabled:
        if a.get("always") or first_match(a.get("paths") or [], changed):
            selected.append(a)
        else:
            dropped.append(a["name"])
    if dropped:
        print(f"affected-only: {len(selected)} of {len(enabled)} leg(s) "
              f"selected from {len(changed)} changed file(s); skipping "
              + ", ".join(dropped), file=sys.stderr)
    return selected


def cmd_json(matrix: dict, changed_from: str | None = None) -> int:
    changed = read_changed(changed_from) if changed_from else None
    include = [
        {
            "name": a["name"],
            "suite": a["suite"],
            "labels": a["labels"],
            "secret_groups": a.get("secret_groups") or [],
        }
        for a in select_areas(matrix, changed)
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


# (changed paths, expected leg names or ALL) for the resolver. Guards the two
# rules whose silent failure costs coverage: fail open, and a shared-machinery
# change running every leg. See cmd_selftest.
ALL = None  # in the table below: expect every enabled leg
SELFTEST_CASES: list[tuple[list[str], set[str] | None]] = [
    # A provider change runs that provider plus the always-on floor.
    (["providers/v1/vault/client.go"], {"core-smoke", "vault"}),
    (["e2e/suites/provider/cases/aws/secretsmanager.go"],
     {"core-smoke", "aws"}),
    # grafana.go selects the generator leg too: both legs run the same suite
    # binary from the same Go package, so a change here can break its compile.
    (["e2e/suites/generator/grafana.go"],
     {"core-smoke", "generator", "grafana"}),
    # Two providers at once select both.
    (["providers/v1/vault/x.go", "providers/v1/gcp/y.go"],
     {"core-smoke", "vault", "gcp"}),
    # Shared machinery runs everything, even though most areas do not list it.
    (["runtime/reconciler.go"], ALL),
    (["apis/externalsecrets/v1/types.go"], ALL),
    (["e2e/framework/util.go"], ALL),
    (["e2e/matrix.yaml"], ALL),
    ([".github/workflows/e2e-reusable.yml"], ALL),
    # Every leg runs the same image and cluster, so these are shared too. They
    # were missed once; a change here skipping the vault leg is the exact
    # coverage loss affected-only selection must never cause.
    # Every provider leg compiles this bootstrap, and it sits above their
    # cases/<name>/** globs, so nothing else would select it.
    (["e2e/suites/provider/suite_test.go"], ALL),
    (["e2e/Dockerfile"], ALL),
    (["e2e/entrypoint.sh"], ALL),
    (["e2e/k8s/vault.values.yaml"], ALL),
    (["e2e/kind.yaml"], ALL),
    # Found by auditing each leg's imports against its globs: these four
    # dependencies are real but not obvious from the leg's name.
    (["e2e/suites/generator/testcase.go"],
     {"core-smoke", "generator", "grafana"}),
    (["e2e/suites/provider/cases/fake/fake.go"],
     {"core-smoke", "flux", "argocd"}),
    (["providers/v1/kubernetes/client.go"], {"core-smoke"}),
    (["providers/v1/fake/fake.go"], {"core-smoke"}),
    # Unrelated changes still run the floor, never an empty matrix. e2e docs
    # are deliberately not shared machinery.
    (["docs/introduction/faq.md"], {"core-smoke"}),
    (["e2e/README.md"], {"core-smoke"}),
    # A near miss must not match: awsx is not aws.
    (["providers/v1/awsx/client.go"], {"core-smoke"}),
    # Fail open: nothing to go on means run everything.
    ([], ALL),
]


def cmd_selftest(matrix: dict) -> int:
    """Exercise select_areas against SELFTEST_CASES. Runs in prepare-matrix
    beside check, so a regression in the resolver fails the build rather than
    quietly shrinking the fan-out."""
    every = {a["name"] for a in matrix["areas"] if a.get("enabled")}
    failures = 0

    def fail(what: str, detail: str) -> None:
        nonlocal failures
        failures += 1
        print(f"FAIL {what}\n  {detail}", file=sys.stderr)

    for changed, expected in SELFTEST_CASES:
        want = every if expected is None else expected
        # select_areas narrates its decision on stderr; capture it so a real
        # failure is not buried, and so the narration can be asserted on.
        log = io.StringIO()
        with redirect_stderr(log):
            got = {a["name"] for a in select_areas(matrix, changed or None)}
        if got != want:
            fail(f"changed={changed}",
                 f"want {sorted(want)}\n  got  {sorted(got)}")
        # A wrongly skipped leg is diagnosed from this log, so it has to name
        # the legs it dropped, or the reason a change ran everything.
        if changed and got != every and "skipping" not in log.getvalue():
            fail(f"changed={changed}", "narrowed the matrix, logged no why")
        if changed and got == every and "full matrix" not in log.getvalue():
            fail(f"changed={changed}", "ran every leg without logging why")

    # These two own the remaining fail-open rules and select_areas never
    # reaches them, so exercise them directly rather than trusting them.
    with redirect_stderr(io.StringIO()):
        cases = [
            ("unreadable file", read_changed(str(HERE / "no-such-file")), None),
            ("blank text", parse_changed("  \n\t\n", "<test>"), None),
            ("two paths", parse_changed("a/b.go\n c/d.go \n", "<test>"),
             ["a/b.go", "c/d.go"]),
        ]
    for what, got_paths, want_paths in cases:
        if got_paths != want_paths:
            fail(f"read_changed/parse_changed on {what}",
                 f"want {want_paths}, got {got_paths}")

    total = len(SELFTEST_CASES) + len(cases)
    if failures:
        print(f"ERROR: {failures} of {total} selftest assertion(s) failed",
              file=sys.stderr)
        return 1
    print(f"matrix.py selftest ok: {total} cases")
    return 0


def main() -> int:
    argv = sys.argv[1:]
    cmd = argv[0] if argv else "check"
    others = {"check": cmd_check, "plan": cmd_plan, "selftest": cmd_selftest}
    changed_from = None
    if cmd == "json":
        if len(argv) > 1:
            if argv[1] != "--changed" or len(argv) != 3:
                print(USAGE, file=sys.stderr)
                return 2
            changed_from = argv[2]
    elif len(argv) > 1 or cmd not in others:
        print(USAGE, file=sys.stderr)
        return 2
    matrix = load_yaml(MATRIX)
    if cmd == "json":
        return cmd_json(matrix, changed_from)
    return others[cmd](matrix)


if __name__ == "__main__":
    sys.exit(main())
