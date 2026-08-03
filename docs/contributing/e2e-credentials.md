# Setting up e2e credentials

Most e2e legs run entirely in kind against an in-cluster backend and need no
credentials at all. A few talk to a real vendor account. Those legs carry a
`secret_groups` entry in `e2e/matrix.yaml`, run only when a maintainer issues
`/ok-to-test`, and stay disabled until someone provisions the account.

This page records how to obtain and scope the credentials for the providers
whose suites exist but do not yet run. It is a reference for whoever picks that
up; it is not part of the user documentation.

## How a credential reaches the suite

Three places have to agree on the variable name, and nothing checks that they
do. A mismatch is silent: the suite reads an empty string and fails somewhere
far from the cause.

1. The secret is stored on the repository (or on the organisation, scoped to
   this repository).
2. `.github/workflows/e2e-reusable.yml` maps it to an env var, gated on the
   leg's `secret_groups`, so a leg only ever sees its own provider's
   credentials.
3. `e2e/run.sh` forwards that env var into the test pod. This is an explicit
   allowlist; a variable missing here never reaches the suite even when it is
   set on the runner.

The suite then reads it with `os.Getenv`. `make -C e2e matrix.plan` prints which
variables each enabled leg receives, without reading any value.

## Akeyless

| Variable | Meaning |
| --- | --- |
| `AKEYLESS_ACCESS_ID` | Access ID of the auth method, `p-...` |
| `AKEYLESS_ACCESS_TYPE` | `api_key` for the setup below |
| `AKEYLESS_ACCESS_TYPE_PARAM` | The access key for that auth method |
| `AKEYLESS_PATH_PREFIX` | Optional. Runs the suite under one folder |

Akeyless has a free tier that is sufficient. Create an API key auth method, a
role with item capabilities, and associate the two:

```bash
akeyless auth-method create api-key --name /eso-e2e
# prints the Access ID and Access Key; the key is shown once

akeyless create-role --name eso-e2e-role
akeyless set-role-rule --role-name eso-e2e-role \
  --path '/eso-e2e/*' --rule-type item-rule \
  --capability read --capability create --capability update \
  --capability delete --capability list
akeyless assoc-role-am --role-name eso-e2e-role --am-name /eso-e2e
```

The suite names its items at the account root by default, which would need a
rule on `/*`. Set `AKEYLESS_PATH_PREFIX` to keep the role scoped to the one
folder above:

```bash
export AKEYLESS_ACCESS_ID=p-...
export AKEYLESS_ACCESS_TYPE=api_key
export AKEYLESS_ACCESS_TYPE_PARAM=...
export AKEYLESS_PATH_PREFIX=/eso-e2e

make -C e2e test.run TEST_SUITES=provider GINKGO_LABELS="akeyless && !managed"
```

Note that authorization is evaluated before existence, so a role that is missing
a capability reports `401 UnauthorizedAccess` rather than a permission-specific
error, and a genuinely absent item reports `404 NotFound` only to a caller
allowed to know that.

Two runs can share one account and one prefix safely as things stand. Item names
derive from the test namespace, which the API server generates, so every spec
gets a distinct name even though ginkgo runs specs in parallel, and no spec
enumerates the account. That last point is what makes it safe: if a spec that
finds by name or tag is ever added, it would match items belonging to a
concurrent run, and runs would need distinct prefixes from then on.

## GitLab

| Variable | Meaning |
| --- | --- |
| `GITLAB_TOKEN` | Token with `api` scope on the project |
| `GITLAB_PROJECT_ID` | Numeric ID of the project the suite writes to |
| `GITLAB_ENVIRONMENT` | Environment scope for the variables it creates |

A free gitlab.com account is sufficient. The suite creates and deletes
**project CI/CD variables** (`ProjectVariables.CreateVariable`), so use a
throwaway project rather than one that matters, and a token scoped to it:

1. Create an empty project. Its numeric ID is on the project overview page.
2. Create a project access token (or a personal token limited to that project)
   with the `api` scope.
3. Create the environment named in `GITLAB_ENVIRONMENT`, or set it to an
   existing one such as `*`.

```bash
export GITLAB_TOKEN=glpat-...
export GITLAB_PROJECT_ID=12345678
export GITLAB_ENVIRONMENT='*'

make -C e2e test.run TEST_SUITES=provider GINKGO_LABELS="gitlab && !managed"
```

## Oracle

| Variable | Meaning |
| --- | --- |
| `OCI_TENANCY_OCID` | Tenancy OCID |
| `OCI_USER_OCID` | User OCID the API key belongs to |
| `OCI_REGION` | Region identifier, for example `uk-london-1` |
| `OCI_FINGERPRINT` | Fingerprint of the uploaded API key |
| `OCI_PRIVATE_KEY` | PEM private key matching that fingerprint |

An OCI account is needed. Check the current always-free limits before relying on
them: the suite uses the Vault service, whose secret storage is not necessarily
covered.

In the OCI console, under your user's API keys, add a key pair. The console
shows the fingerprint and offers the private key for download, and prints a
configuration snippet containing the tenancy, user and region OCIDs.

**This suite cannot run as currently written.** Two things need fixing first:

- The names do not match. `e2e-reusable.yml` and `run.sh` supply `ORACLE_USER_OCID`,
  `ORACLE_TENANCY_OCID`, `ORACLE_REGION`, `ORACLE_FINGERPRINT` and `ORACLE_KEY`,
  while `suites/provider/cases/oracle/provider.go` reads the `OCI_*` names in
  the table above. Nothing bridges them, so every value arrives empty.
- The vault is hardcoded. The `SecretStore` is built with
  `Vault: "vaultOCID"`, a placeholder rather than a real OCID, and no variable
  exists to supply one.

Anyone enabling this leg should expect to fix both, and to add whichever
variable ends up carrying the vault OCID to all three places listed at the top
of this page.
