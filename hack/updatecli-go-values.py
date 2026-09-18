#!/usr/bin/env python3

# Copyright 2026 The External Secrets authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Build deduplicated Updatecli inputs from the repository's Go modules."""

import json
import os
import re
import subprocess
import sys
from pathlib import Path

INTERNAL_MODULE = "github.com/external-secrets/external-secrets"
PSEUDO_VERSION = re.compile(r"^v\d+\.\d+\.\d+-\d{14}-[0-9a-f]+$")
SEMVER_MAJOR = re.compile(r"^v?(\d+)\.")


def parse_go_mod(path: Path) -> dict[str, tuple[str, bool]]:
    environment = os.environ.copy()
    environment["GOWORK"] = "off"
    result = subprocess.run(
        ["go", "-C", str(path.parent), "mod", "edit", "-json"],
        check=True,
        capture_output=True,
        env=environment,
        text=True,
    )
    go_mod = json.loads(result.stdout)
    requirements = {
        requirement["Path"]: requirement
        for requirement in go_mod.get("Require") or []
    }
    direct = {
        module
        for module, requirement in requirements.items()
        if not requirement.get("Indirect", False)
    }
    tool_modules: set[str] = set()
    missing_tools = []
    for tool_spec in go_mod.get("Tool") or []:
        tool = tool_spec["Path"]
        candidates = [
            module
            for module in requirements
            if tool == module or tool.startswith(f"{module}/")
        ]
        if not candidates:
            missing_tools.append(tool)
            continue
        tool_module = max(candidates, key=len)
        tool_modules.add(tool_module)

    if missing_tools:
        missing = ", ".join(sorted(missing_tools))
        raise ValueError(f"{path}: tool packages missing from require directives: {missing}")

    result = {
        module: (requirements[module]["Version"], False)
        for module in direct
    }
    for module in tool_modules:
        result.setdefault(module, (requirements[module]["Version"], True))
    return result


def version_filter(version: str) -> tuple[str, str]:
    if PSEUDO_VERSION.match(version):
        return "latest", ""

    match = SEMVER_MAJOR.match(version)
    if not match:
        raise ValueError(f"unsupported Go module version {version!r}")
    if "-" in version:
        return "semver", f"{match.group(1)}.x.x-0"
    return "semver", f"{match.group(1)}.x"


def main() -> None:
    # A source is shared by every go.mod using the same module major. Native
    # targets then write that one resolved version to every direct occurrence.
    dependencies: dict[tuple[str, str, str], dict[str, object]] = {}

    module_files = sorted(
        path
        for path in Path(".").rglob("go.mod")
        if not {".git", "bin", "node_modules", "vendor"}.intersection(path.parts)
    )
    for path in module_files:
        for module, (version, file_target) in parse_go_mod(path).items():
            if module == INTERNAL_MODULE or module.startswith(f"{INTERNAL_MODULE}/"):
                continue

            kind, pattern = version_filter(version)
            key = (module, kind, pattern)
            dependency = dependencies.setdefault(
                key,
                {
                    "module": module,
                    "kind": kind,
                    "pattern": pattern,
                    "files": {},
                },
            )
            files = dependency["files"]
            assert isinstance(files, dict)
            files.setdefault(
                path.as_posix(),
                {
                    "path": path.as_posix(),
                    "filetarget": file_target,
                    "matchpattern": rf"(?m)^(\s*){re.escape(module)}\s+\S+(\s+// indirect)$",
                },
            )

    output = []
    for key in sorted(dependencies):
        dependency = dependencies[key]
        files = dependency["files"]
        assert isinstance(files, dict)
        dependency["files"] = [files[path] for path in sorted(files)]
        output.append(dependency)

    json.dump(
        {
            "go": {
                "dependencies": output,
                "files": [path.as_posix() for path in module_files],
            }
        },
        fp=sys.stdout,
        indent=2,
    )
    print()


if __name__ == "__main__":
    main()
