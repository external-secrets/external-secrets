# License Header Verification Tools

This directory contains scripts to verify that source code files have the correct Apache License 2.0 headers.

## Scripts

### verify-license-header.sh

Checks license headers for all Go files in the repository or specific files passed as arguments.

Usage:
```bash
# Check all Go files
./hack/verify-license-header.sh

# Check specific files
./hack/verify-license-header.sh pkg/provider/example/provider.go
```

### check-pr-license-headers.sh

Checks license headers only for files added in the current pull request. This is used by the CI system to ensure new files have proper license headers.

Usage:
```bash
# Check files added in PR against main branch
./hack/check-pr-license-headers.sh

# Check files added in PR against specific branch
./hack/check-pr-license-headers.sh origin/develop
```

## Makefile Targets

- `make license-check`: Check license headers in all Go files
- `make license-check-pr`: Check license headers in files added to current PR

## License Template

The license header template is stored in `hack/boilerplate.go.txt`. All Go files should start with this header (wrapped in `/* */` comment block).

## Integration

The license check is automatically run as part of the CI pipeline for pull requests. If any newly added Go files are missing the license header, the CI will fail.