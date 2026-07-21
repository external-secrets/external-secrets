# Filtering Keys with DataFrom Select

When calling out an ExternalSecret with `dataFrom.extract` or `dataFrom.find` to retrieve multi-key secrets, External Secrets Operator automatically synchronizes every key-value pair returned by the provider into the target Kubernetes Secret.

However, many secret managers return metadata or helper fields alongside the actual sensitive credentials. For example, 1Password includes default fields such as `notesPlain`, `username`, and `password` even if they are empty or completely unused in the application.

`dataFrom.select` provides a solution to filter and shape the returned keys **after** they are fetched (and after any [`dataFrom.rewrite`](datafrom-rewrite.md) operations). Each select rule matches keys using exact `names` and/or a `regexp`, applying sequential `Include` or `Exclude` operations to produce a clean target secret.

## How it works

Processing order for each `dataFrom` entry:

1. Fetch keys from the provider (`extract`, `find`, or generator).
2. Apply `rewrite` (if any).
3. Apply `select` operations in order.
4. Continue with conversion / templating / write to the Kubernetes Secret.

Rules of thumb:

* If `select` is omitted, all keys are kept.
* Each select entry must specify at least one of `regexp` or a non-empty `names` list.
* Matching uses the **current** key names (post-rewrite).
* An invalid `regexp` puts the ExternalSecret into an error state.

## How-to: drop unwanted 1Password fields

Consider the following scenario:
> "You are synchronizing secrets from 1Password to Kubernetes using External Secrets Operator. However, 1Password includes unnecessary fields by default, such as `notesPlain`, `username`, and `password`. These fields are synchronized even when they are blank, resulting in cluttered Kubernetes Secrets."

### Without select

Suppose the provider returns the following keys:
```text
host: db.example.com
port: "5432"
password: secret
username: ""
notesPlain: ""
```

Without any select rules, synchronizing these keys directly with `dataFrom.extract` produces a cluttered target Kubernetes Secret containing unnecessary empty/blank fields:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: database-credentials
type: Opaque
data:
  host: ZGIuZXhhbXBsZS5jb20=        # db.example.com
  port: NTQzMg==                    # 5432
  password: c2VjcmV0                # secret
  username: ""                      # synchronized even though it is blank
  notesPlain: ""                    # synchronized even though it is blank
```

### The Solution using `dataFrom.select`

To resolve this and keep the target Kubernetes Secret clean, it is possible to define a list of `select` rules under `dataFrom` to exclude the unwanted blank fields:

```yaml
{% include 'datafrom-select-1password.yaml' %}
```

Applying the ExternalSecret with the `Exclude` select rule results in a clean Kubernetes Secret containing only the active, desired keys:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: database-credentials
type: Opaque
data:
  host: ZGIuZXhhbXBsZS5jb20=        # db.example.com
  port: NTQzMg==                    # 5432
  password: c2VjcmV0                # secret
```

### How it works

The `Exclude` operation matches `username` and `notesPlain` from the names list and drops them from the set of keys after they are fetched.

If an allow-list approach is preferred instead of specifying which fields to drop, `Include` rules can be used to keep only the specific keys, dropping all others by default.

## Include / Exclude semantics

To work effectively with `select`, it is important to understand how operations modify the state of the secret keys.

### Sequential Map Transform

The select process can be understood as a pipeline operating on a **working set** of keys (a map):
1. **Initial State:** The working set starts with all keys retrieved from the secret provider (after any `rewrite` operations).
2. **Sequential Execution:** Select rules are applied one after another in the exact order they are defined in the YAML.
3. **In-place Transformation:**
   * **`Exclude`** removes matched keys from the working set.
   * **`Include`** behaves as an **intersection/filter**: it keeps only the matched keys that are *currently* in the working set, dropping all other remaining keys.
4. **Final State:** The keys remaining in the working set after the last operation are written to the Kubernetes Secret.

!!! danger "Crucial Insight: `Include` does not restore keys"
    An `Include` operation **never merges keys back** from the original provider payload. If a key was removed by a prior `Exclude` (or not kept by a prior `Include`), a subsequent `Include` matching that key will have no effect. Once a key is gone, it cannot be brought back by later operations in the sequence.

| Operation | Effect on the current key set |
|-----------|-------------------------------|
| `Exclude` | Remove matched keys |
| `Include` | Keep only matched keys (drop everything else still in the set) |

### Detailed Walkthroughs

#### Case 1: Exclude then Include All

In this scenario, a specific key is excluded and then all remaining keys are included using a wildcard regular expression `.*`.

```yaml
{% include 'datafrom-select-exclude-then-include.yaml' %}
```

**Walkthrough:**
In this example, assume the provider initially returns three keys: `{host, port, password}`.

1. **Step 1: Exclude `host`**
   * The operator matches `host` and removes it.
   * The working set becomes: `{port, password}`.
2. **Step 2: Include `.*`**
   * The operator applies the regex `.*` (which matches all strings) to the *current* working set `{port, password}`.
   * Since `host` was already deleted in Step 1, the wildcard `.*` only sees and keeps `{port, password}`. It does not restore `host` from the provider's original payload.
   * **Result:** `{port, password}`.

---

#### Case 2: Include then Exclude

In this scenario, keys are filtered to keep only those matching a prefix, and then a specific key is excluded from that subset.

```yaml
{% include 'datafrom-select-include-then-exclude.yaml' %}
```

**Walkthrough:**
In this example, assume the provider initially returns: `{db_host, db_port, db_password, app_name}`.

1. **Step 1: Include `^db_`**
   * The operator intersects the working set with the pattern `^db_`. Only `db_host`, `db_port`, and `db_password` match.
   * `app_name` does not match, so it is dropped.
   * The working set becomes: `{db_host, db_port, db_password}`.
2. **Step 2: Exclude `db_password`**
   * The operator matches `db_password` and removes it from the current working set.
   * **Result:** `{db_host, db_port}`.

---

### Step-by-Step Edge Cases Walkthrough

To further clarify these semantics, the following traces demonstrate how the operator handles various edge-case configurations on an initial working set of `{host, port, password}` and other keys starting with `foo-` or `bar-`.

| Sequence | Result | Why |
|----------|--------|-----|
| `Exclude .*` then `Include .*` | **Empty** | Exclude removes every key; Include only filters what is left |
| `Include .*` then `Exclude .*` | **Empty** | Include keeps everything, then Exclude removes everything |
| `Exclude ^foo-` then `Include ^bar-` | Only remaining `bar-*` keys | Include never restores previously excluded `foo-*` keys |
| `Exclude host` then `Include .*` | All keys except `host` | Include applies to the set after Exclude |

#### `Exclude .*` then `Include .*`
1. **Initial Set:** `{host, port, password}`
2. **Step 1 (`Exclude .*`):** The regex `.*` matches and removes all keys. The working set becomes empty (`{}`).
3. **Step 2 (`Include .*`):** The operator attempts to filter the empty working set. No keys exist to match against `.*`.
4. **Final Result:** **Empty** (no keys are synchronized).

#### `Include .*` then `Exclude .*`
1. **Initial Set:** `{host, port, password}`
2. **Step 1 (`Include .*`):** The regex `.*` matches and keeps all keys. The working set remains `{host, port, password}`.
3. **Step 2 (`Exclude .*`):** The regex `.*` matches and removes all remaining keys.
4. **Final Result:** **Empty** (no keys are synchronized).

#### `Exclude ^foo-` then `Include ^bar-`
1. **Initial Set:** `{foo-key1, foo-key2, bar-key1, bar-key2, other-key}`
2. **Step 1 (`Exclude ^foo-`):** The regex matches and removes all keys starting with `foo-`. The working set becomes `{bar-key1, bar-key2, other-key}`.
3. **Step 2 (`Include ^bar-`):** The operator filters the current working set to keep only keys starting with `bar-`. Thus, `other-key` is dropped.
4. **Final Result:** `{bar-key1, bar-key2}`. Previously excluded `foo-` keys are never restored.

#### `Exclude host` then `Include .*`
1. **Initial Set:** `{host, port, password}`
2. **Step 1 (`Exclude host`):** The operator removes `host`. The working set becomes `{port, password}`.
3. **Step 2 (`Include .*`):** The operator filters the working set, keeping everything matching `.*`. This keeps `{port, password}`.
4. **Final Result:** `{port, password}` (all keys except `host` are kept).

### Combining regexp and names

A single select entry may set both `regexp` and `names`. Keys that match either matcher are treated as matched for that operation.

## Using select with rewrite

When both are set, rewrite runs first. If `path/to/password` is rewritten to `password`, `select` rules must match `password`, not the original path.
