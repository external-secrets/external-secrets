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
| `GITLAB_ENVIRONMENT` | Read into the credentials Secret, but unused. Set it to `*` |

A free gitlab.com account is sufficient. The suite creates and deletes **project
CI/CD variables** (`ProjectVariables.CreateVariable`), so point it at a
throwaway project rather than one that matters. Nothing needs to be created
inside the project beforehand; the suite makes and removes every variable
itself.

1. Create a group, then create the throwaway project **inside that group**. The
   project alone is enough for the suite as it stands, but the provider also
   supports resolving variables from a project's groups, and covering that needs
   a group to hang variables on. Starting with one costs nothing and saves
   moving the project later.
2. Take the numeric project ID from the project overview page, under the project
   name.
3. Create a personal access token with the `api` scope. A project access token
   scoped to the one project is tighter if your plan offers it. Either way the
   identity needs at least Maintainer on the project, which you have if you
   created it.
4. Leave `GITLAB_ENVIRONMENT` as `*`. The suite reads it into the credentials
   Secret, but it never reaches the `SecretStore` and variables are created with
   a nil environment scope, so it changes nothing.

There is no URL variable. `GitlabProvider.URL` is only applied when non-empty
(`providers/v1/gitlab/provider.go:93`), so an unset value uses gitlab.com.

```bash
export GITLAB_TOKEN=glpat-...
export GITLAB_PROJECT_ID=12345678
export GITLAB_ENVIRONMENT='*'

make -C e2e test.run TEST_SUITES=provider GINKGO_LABELS="gitlab && !managed"
```

Ginkgo runs specs in parallel, so this is several concurrent variable
create/delete cycles against one project, and GitLab rate-limits. Project
variables are also a flat namespace per project with no prefix equivalent, so do
not point two runs at the same project at once.

## Oracle

An always-free OCI account is sufficient. Every tenancy gets 150 Always Free Vault
secrets, and master encryption keys protected by software are free, which covers
everything this suite needs.

**This suite cannot run as currently written**, so the setup below prepares the
tenancy but will not yet produce a passing run. See the gaps at the end of this
section.

### What to create in the tenancy

**A vault.** Identity & Security, then Vault, then Create Vault. Leave the virtual
private vault option unticked; that variant is billed. Creation takes several
minutes. Reuse one vault rather than making throwaways: deleting a vault is
scheduled a minimum of seven days out and it stays in the tenancy meanwhile.

**A master encryption key inside that vault.** Open the vault, then Master Encryption
Keys, then Create Key. Protection Mode `Software` (HSM-protected keys give only 20
free key versions), algorithm `AES`, length 256 bits. The key must be symmetric and
must live in the vault above: OCI rejects asymmetric keys for secrets and rejects a
key from a different vault.

```bash
oci kms management key create \
  --compartment-id <compartment OCID> \
  --display-name eso-e2e \
  --protection-mode SOFTWARE \
  --key-shape '{"algorithm":"AES","length":32}' \
  --endpoint <the vault's management endpoint>
```

The endpoint is per-vault, shown on the vault's detail page, and the CLI takes the
AES length in bytes rather than bits.

**An API signing key.** Under your user's settings, add an API key. Let the console
generate the pair: the key must have **no passphrase**, because the suite passes
`nil` in the passphrase slot of `common.NewRawConfigurationProvider`. The console
shows the fingerprint, offers the private key for download, and prints a
configuration snippet containing the tenancy, user and region OCIDs.

**An IAM policy**, unless you are the tenancy administrator, in which case you
already have this:

```
Allow group <group> to manage secret-family in compartment <name>
Allow group <group> to read vaults in compartment <name>
Allow group <group> to read keys in compartment <name>
```

`read vaults` is required even though the suite never reads the vault itself.
ESO's own `newClient` calls `KmsVaultClient.GetVault` when it builds the client
(`providers/v1/oracle/oracle.go`), so the store fails validation without it.

If the vault sits in the root compartment, its compartment OCID and the tenancy
OCID are the same value.

### Variables

| Variable | Meaning |
| --- | --- |
| `ORACLE_TENANCY_OCID` | Tenancy OCID |
| `ORACLE_USER_OCID` | User OCID the API key belongs to |
| `ORACLE_REGION` | Region identifier, for example `uk-london-1` |
| `ORACLE_FINGERPRINT` | Fingerprint of the uploaded API key |
| `ORACLE_KEY` | PEM private key matching that fingerprint, contents not a path |

```bash
export ORACLE_TENANCY_OCID=ocid1.tenancy.oc1..
export ORACLE_USER_OCID=ocid1.user.oc1..
export ORACLE_REGION=uk-london-1
export ORACLE_FINGERPRINT=..
export ORACLE_KEY="$(cat ~/.oci/eso-e2e.pem)"
```

### Quota

Deleting a secret is a scheduled operation, and `timeOfDeletion` defaults to **30 days**
out. The accepted range is 1 to 30 days. A secret pending deletion keeps both its name
and its slot, so leaving the default in place means a suite run's secrets occupy the
tenancy for a month. At roughly 15 to 20 secrets per run that exhausts the 150 always
free secrets in about seven runs, after which creates start failing for a reason that
looks nothing like a quota problem.

Anything creating secrets here should schedule deletion at the 1 day minimum.

Every state transition in the Vault service is also asynchronous, and the next operation
is rejected with `409 IncorrectState` until the previous one settles. Creating then
immediately deleting fails, because the secret is still `CREATING`; cancelling a pending
deletion then immediately rescheduling fails, because it is still `CANCELLING_DELETION`.
Measured latencies are a few seconds each, but they are not zero and they are not
bounded by anything documented.

### Why it does not run yet

Tracked in
[#6767](https://github.com/external-secrets/external-secrets/issues/6767). Two of
the gaps affect the variables on this page:

- **The names disagree.** `e2e-reusable.yml` and `run.sh` supply the `ORACLE_*`
  names above, while `suites/provider/cases/oracle/provider.go` reads `OCI_TENANCY_OCID`,
  `OCI_USER_OCID`, `OCI_REGION`, `OCI_FINGERPRINT` and `OCI_PRIVATE_KEY`. Nothing
  bridges them, so every value arrives empty.
- **Three OCIDs have no variable at all.** Creating a secret through the Vault API
  requires the vault, compartment and encryption key OCIDs, and the `SecretStore`
  requires the vault OCID, which is currently hardcoded to the placeholder
  `"vaultOCID"`. Whatever names those end up taking have to be added to all three
  places listed at the top of this page.

The rest of the gaps are in the suite's use of the OCI API rather than in its
configuration. Anyone picking this up should read #6767 first rather than assuming
the two points above are the whole of it.
