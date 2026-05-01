# Sprig Template Functions (Vendored)

This package contains template functions copied from
[github.com/Masterminds/sprig/v3](https://github.com/Masterminds/sprig)
(v3.3.1-0.20241028115027-8cb06fe3c8b0).

The original code is licensed under the MIT License. See the upstream repository
for the full license text.

The following functions have been intentionally excluded:

- `env` — exposes environment variables to templates
- `expandenv` — expands environment variables in strings
- `getHostByName` — performs DNS lookups

These functions were removed to prevent information disclosure through templates.
