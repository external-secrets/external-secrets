# Template cleaning and explicit public error information

**Status:** Proposed implementation plan. This document does not implement the Go
changes, and its example tests and commands have not yet been run.

## 1. Purpose and agreed decisions

Prevent fetched secrets, sensitive template source, generated keys, and rejected
values from appearing in Kubernetes Events and status through error messages.
Protect the controller logging/return paths involved in the same failures. Remove
source of explicit arbitrary-error primitives (like `fail`) that would report
bad information. However, this recognizes that we can't delete everything.

The design principle is:

> Producers (sdks, providers) explicitly describe what is public (the error information that will be published);
> Consumers (controllers) always filter before publishing;
> Publishing (to k8s api objects, logs, or events) is santized by default;
> Preserve all error information/original causes internally for filtering

Here, **sanitize means select only "classified as public" information and
suppress all unclassified information**.
It does not mean searching an arbitrary error string for known passwords and replacing them.

### Golden rules for error text management

1. **Unknown errors are private by default.** An SDK/API/library error does not
   become public because its origin looks trustworthy.
2. **Public text is a producer declaration.** Store it separately from the cause;
   do not approve the cause's entire `Error()` text implicitly. Producers are
   free to use whatever they want as public text. **We do not judge in core**.
3. **One error mechanism.** Evolve the existing `Safe`/`SafeMessage` mechanism into
   `runtime/errinfo`; do not add a parallel renderer reporting framework.
4. **Use `Info`, `Failure`, and `NewFailure`.** We do not want to introduce a new
   provider interface like `SafeMessage()`. This would mean a callback.
   In other words: no callbacks, universal safe-error interfaces or "authoritative"
   (=common in our codebase) error sentinel values that our providers should use.
5. **`Failure.Error()` is safe by default and simple.** Return this failure's
   declared public message, or a default fallback. **Do not traverse its cause**
6. **Keep standard `Unwrap()`.** Unwrap() will unwrap causes and cause may be sensitive;
   it exists for internal recognition in the controller only.
   This allows us to continue to use Go error traversal tooling, while keeping the
   public reasons of any error very explicit and simple.
7. **Consumers filter even when `Error()` is safe.**
   The controllers receive the full cause/error (like provider' SDK failures)
   and can produce safe text with Error().
   It also sees _where_ the error happens, and can decide what to filter/override.
   It is the better place to guarantee consistency in implementation and behaviour.
8. **The first _classified_ `Failure` controls public detail.** A plain `%w` wrapper
   contributes no classified text, it's only contributing to the errors/causes.
   As said before, an outer `Failure` with no public message selects the consumer/caller's fallback.
   Therefore, the first _failure with classified text_ will be the public output
   of Error()
9. **Add context explicitly through a helper.** Combine controller-owned context
   with already-filtered detail, never by manipulating/concatenating raw `err.Error()`.
10. **Separate display from behavior.** Make conflict/deletion/retry decisions of
    errors depend on the internal error, not on public text. This keeps the
    existing behaviour and is consistent with go expectations. We just wrap
    by defining what's actually publishable.
11. **Public projection contains no private cause.** Events/status receive a
    string; error-consuming sinks receive a new error containing only that string
    (using a helper tool).
12. **When possible, identify _sources of error_, not _content of errors_**.
    For example, we can use something like a new `TemplateRef.SpecPath`
    to identify where the template configuration comes from, but not have
    its content. It would be enough for a user to debug, while being safe.
    In other words, never derive public text from fetched values,
    template text, generated keys, or arbitrary nested error messages.
13. **Make this extensible to allow new capabilities only when the time arises.**
    Future behavior metadata/interfaces are possible,
    but this design does not invent `Retryable()` as an interface for a Failure
    type. It does not replace existing Kubernetes predicates and sentinels.
    In other words: Stay close to what we currently do.
14. **Make this a new capability, not a breaking change**. No interface change
    for existing providers. Just make it possible for providers to classify
    their error outputs.

In addition to that, here are a few rules for template management.

1. **Render first, apply the result separately.** `Values` and `KeysAndValues`
    describe output application, not fundamentally different template programs.
2. **Keep template structures small.** Use `TemplateRef` and `Template`; no
    `TemplateRoot`/`StepKind` hierarchy, universal `TemplateEntry`, or parallel
    Values/KeysAndValues request trees.
3. **Inline key disclosure is an explicit policy decision.** Initially report
    the containing spec field, not a configured map key. Do not assume every key
    the engine receives is safe.
    This changes the current logging behaviour for a safer alternative.

### (Pseudo-) Practical case

For this conversation, let's assume a template author
(ExternalSecrets create/update) can read Events but cannot read the output Secret
(this is an odd rbac, but let's accept it for a moment).

Error publication must not bridge that permission gap.

A template can output information through success (k8s resources - objects,
annotations, ...)
or failures (k8s events). We do not want this output information to be
a leak of confidential data.

Restricting templates to ExternalSecrets authors while keeping usefulness is
impractical (admission controller on some parts of the custom resource is
tedious).

## Current templating risks

The current path is:

```text
ExternalSecret Reconcile
  GetProviderSecretData
  createSecret / updateSecret
    mutationFunc -> ApplyTemplate -> Parser.MergeMap
      v2.Execute -> valueScopeApply -> execute
        template receives plaintext map[string]string as dot
        fail function (or whatever other function failing) returns errors.New(rendered argument)
  markAsFailed -> Event(err.Error())
  return original error -> controller-runtime logging
```

`markAsFailed` in `pkg/controllers/externalsecret/externalsecret_controller.go`
separately calls `ctrlutil.SafeMessage` for the Ready condition. The Event
has already received the unfiltered error, while the condition is safe.
Existing `markasfailed_test.go` tests protect conditions but do not assert Event contents.

The original example needs an actual `data` or `dataFrom` input to expose fetched
values. The webhook rejects specs containing neither. But webhook can be disabled.

Other paths must be included in implementation:

| Path | Current behavior |
| --- | --- |
| `runtime/template/v2/template.go:mapScopeApply` | Calls `execute(tpl, tpl, data)`: complete source text is also the template's name. |
| `pkg/controllers/templating/parser.go` | KeysAndValues maps sometimes use source text as their map key. |
| `renderTemplatedManifest` | Renders Secret/ConfigMap sources into `tempSecret.Data`, then executes those values again; first-pass output becomes second-pass source. |
| `runtime/esutils/utils.go:RewriteTransform` | Executes Go templates directly, bypassing v2 `Execute`, and prints the current key in errors. |
| PushSecret `compileRewrite` / `rewriteWithKeyMapping` | Another direct execution path; outer errors print transformed keys. |
| ExternalSecret `createSecret` / `updateSecret` | Mark arbitrary Kubernetes write errors with `Safe`; API rejections may echo rendered values. |
| ExternalSecret status-update defers | Can log and replace the final returned error after normal control flow finishes. |
| SecretStore `validateStore` | Uses generic conditions but publishes raw provider errors as Events. |
| PushSecret `markAsFailed` | Accepts preformatted strings; several callers interpolate provider errors before calling it. |

## High-level architecture

```text
Producer or adapter
  understands the failing operation
  returns Failure{explicit public Info, original private cause}
                  |
                  v
Controller
  errors.Is / errors.As / Kubernetes predicates determine behavior
  WithContext adds approved operation context when appropriate
                  |
                  v
Publication boundary
  PublicMessage: approved string or caller-owned fallback
  PublicError: same selection, represented as a cause-free error
                  |
                  v
Events / conditions / ordinary logs / controller-runtime
```

The producer can be an ESO provider, the template subsystem, a Kubernetes client
adapter, or controller-owned validation.
It is ESO's team to promote the use of `Failure` for its internal needs.

A producer that has not adopted `Failure` continues returning ordinary errors.
In that case, consumers publish their generic fallback.

There is no prerequisite to update all providers before fixing Events.

### Package ownership

- **`runtime/errinfo`**: evolve the existing marker/extraction machinery into the
  shared `Info`/`Failure` implementation and filtering/context/projection helpers.
  This is a package in the existing runtime Go module, not a new module.
- **`runtime/template` and `runtime/template/v2`**: render execution and output
  application, with template-specific references. They depend on `errinfo`, never
  on controller utilities.
- **`pkg/controllers/util`**: Kubernetes-specific classification and temporary
  compatibility delegates for migrated safe-message helpers.
- **Providers**: classify their own SDK failures and explicitly declare public
  information when useful. They import runtime, not controller packages.
- **Controllers**: make behavioral decisions and use the shared publication
  helpers at every affected sink.

## 4. Shared error contract: `runtime/errinfo`

Add `runtime/errinfo/error.go` and tests by evolving the behavior currently in
`pkg/controllers/util/statuserr.go`. The following API is the implementation
contract; normal package comments/license headers are omitted from snippets.

### 4.1 Representation and constructor

```go
type Info struct {
    // PublicMessage is explicitly approved for Events, status, and ordinary logs.
    // Empty means this failure supplies no public detail.
    PublicMessage string
}

type Failure struct {
    info  Info
    cause error
}

func NewFailure(info Info, cause error) *Failure {
    return &Failure{info: info, cause: cause}
}

func (f *Failure) Error() string {
    if f.info.PublicMessage != "" {
        return f.info.PublicMessage
    }
    return "operation failed"
}

// Unwrap exposes the original cause for internal recognition.
// The cause may contain secret material and must not be published directly.
func (f *Failure) Unwrap() error {
    return f.cause
}
```

`NewFailure` always creates a failure, even when the cause is nil. A nil cause is
useful for local validation that detects a problem without a lower-level error.
Callers wrapping another operation must check `err != nil` before constructing it.
Do not return a typed nil `*Failure` in an `error` interface.

Private fields prevent mutation after declaration. The public message is supplied
as data, not computed by a provider callback when a controller reports the error.
`Error()` formats only this failure; filtering is not done/hidden inside that method.
Do not add generic `Details any` or automatically serialize causes (which are
sensitive) into `Info` (which is less sensitive).

### 4.2 Selecting public information

Convenience function:

```go
func PublicMessage(err error, fallback string) string {
    var failure *Failure
    if errors.As(err, &failure) && failure != nil && failure.info.PublicMessage != "" {
        return limitPublicMessage(failure.info.PublicMessage)
    }
    return limitPublicMessage(fallback)
}
```

`limitPublicMessage` is the existing rune-aware truncation logic moved to the
shared package. Define `MaxPublicMessageLength = 256`; cap the final selected
message, including context. This is a resource bound, not a security filter.
`Failure.Error()` can return its full approved message; outward consumers use the
bounded helper. Empty fallback is supported for context composition.

This code example is just an example and may evolve over time.

A few examples of evolutions:
- We might want to rename `limitPublicMessage` to a more explicit naming (truncatePublicMessage)
- We might want to conditionally truncate (we therefore we need some kind of mutating option
  to the publicMessage).
- We might want to apply other alterations based on needs (EventPublications is not the same
  as resources messages)

Required public information selection behavior:

| Input | Selection |
| --- | --- |
| Raw SDK/API/Go error | Caller fallback; never raw `Error()`. |
| `Failure` with public text | That text. |
| Outer `Failure` around another `Failure` | The outer declaration controls output. |
| Outer `Failure` with an empty message | Fallback, even if an inner failure has public text. |
| `errors.Join` | Standard `errors.As` traversal selects the first matching failure. Do not concatenate child messages; an empty first match selects fallback. |
| Nil error | Fallback, including empty fallback. Callers decide whether a failure exists. |

This deliberately changes the old innermost-marker rule. An arbitrary text
wrapper is still ignored. But a new explicit public declaration is authoritative,
and an empty declaration can withhold inner detail. Add tests documenting this
semantic change rather than porting old marker tests mechanically.

For example:

```go
classified := errinfo.NewFailure(errinfo.Info{
    PublicMessage: "provider authentication failed",
}, sdkError)

err := fmt.Errorf("request used token %s: %w", token, classified)
// PublicMessage(err, "provider request failed") selects only:
// provider authentication failed
```

**The ordinary wrapper is not approved merely because it contains a `Failure`.**
It must be explicitly expressed as Failure.

Neither `PublicMessage` nor `Failure.Error` invokes `Error()` on an unknown cause
to prevent leakage.

### 4.3 Approved context composition

Add the helper for higher-level context:

```go
func WithContext(err error, safeContext string) error {
    if err == nil {
        return nil
    }
    message := safeContext
    if detail := PublicMessage(err, ""); detail != "" {
        if message == "" {
            message = detail
        } else {
            message += ": " + detail
        }
    }
    return NewFailure(Info{PublicMessage: message}, err)
}
```

Keep in mind that this helper is not able to prevent leaks of information.

`context` must itself be safe/approved: an operation constant or a `TemplateRef.SpecPath`
constructed from fixed spec field names and actual list indexes are fine.
Do not pass a formatted SDK error, source text, generated key, or arbitrary target path here.

Examples:

```go
reported := errinfo.WithContext(err, "could not update secret")
// Declared detail present: could not update secret: Kubernetes object validation failed
// No declared detail:      could not update secret
```

Only add context where it adds information. Do not call this helper in every
stack frame or repeat the same operation at both origin and publication. It keeps
the original error as cause so `errors.Is`/`errors.As` still work internally.

### 4.4 Public projection for error-consuming sinks

```go
func PublicError(err error, fallback string) error {
    if err == nil {
        return nil
    }
    return errors.New(PublicMessage(err, fallback))
}
```

Use `PublicMessage` for Events/status and `PublicError` when a logging API or
controller-runtime requires an `error`. The latter is an adapter, not a second
custom error hierarchy. It deliberately has no cause to unwrap.

Do not call `PublicError` in internal provider/helper returns: doing so would lose
SDK identity and controller sentinels too early. Do not repeatedly filter a
projected error: it is now an ordinary cause-free error, so filtering it again
would select a fallback. Project once at each outward sink. A logging projection
must not replace the internal error that control flow is still inspecting.

Safe `Error()` does not authorize dumping the whole `Failure` struct or its cause
with reflection, structured serialization, or a debug formatter. Public sinks
receive the projection, not the internal object graph.

## 5. Template core: execute text, then apply its result

### 5.1 Small template types

Define these types in the `runtime/template` facade:

```go
type TemplateRef struct {
    // Configuration entry that requested rendering.
    // Example: spec.target.template.templateFrom[0].secret.items[1]
    // Construct from spec structure, never template/source/output contents.
    SpecPath string
}

type Template struct {
    Ref TemplateRef

    // Go template program to execute. It may have come from a Secret or
    // an earlier rendering pass and is potentially sensitive.
    Text string
}

type RenderFunc func(Template, map[string][]byte) ([]byte, error)
```

A reference identifies **which configuration entry requested the work**. It is not
an external provider secret identifier or a rendered output key. Every output of
one document template retains that template's reference.

No `DestinationKey` belongs in `Template`. A Values caller already knows the key
it will populate; a KeysAndValues template supplies keys/structure in its output.
Those are application concerns, not attributes of the program being executed.

Initial reference policy:

| Source | Public reference |
| --- | --- |
| ExternalSecret inline data | `spec.target.template.data` |
| ExternalSecret labels/annotations | `spec.target.template.metadata.labels` / `.annotations` |
| Secret item in templateFrom | `spec.target.template.templateFrom[i].secret.items[j]` |
| ConfigMap item in templateFrom | Corresponding `.configMap.items[j]` |
| Literal templateFrom | `spec.target.template.templateFrom[i].literal` |
| PushSecret sources | Same structure rooted at `spec.template` |
| ExternalSecret rewrite operation | `spec.dataFrom[i].rewrite[j]` when known to the caller |
| PushSecret rewrite operation | Actual indexed spec source when available; otherwise fixed `rewrite[j]` context |

Indexes refer to real zero-based spec lists, not Go map iteration. Do not pretend
inline maps have stable entry indexes. Initially omit even inline configured key
names; adding those later requires an explicit disclosure policy.

### 5.2 Avoid an import cycle without another types framework

`runtime/template` already imports v2. Keep the low-level v2 function independent
of facade types:

```go
// runtime/template/v2
func Render(text string, data map[string][]byte) ([]byte, error)
```

Change `runtime/template.EngineForVersion` to return `RenderFunc`. For v2 it returns
a small closure that calls `v2.Render(t.Text, data)` and, on error, applies
`errinfo.WithContext(err, t.Ref.SpecPath)`. Return no partial output on failure.
Unsupported versions get a fixed public declaration, not the supplied version.

Dependency direction remains:

```text
runtime/template -> runtime/template/v2 -> runtime/errinfo
runtime/template ----------------------> runtime/errinfo
runtime/esutils -> runtime/template/v2 + runtime/errinfo
```

No v2 import of its parent facade, no extra `TemplateRoot`/`StepKind` package, and
no controller dependency from runtime are necessary.

### 5.3 Evolve v2 `execute` into the low-level renderer

In `runtime/template/v2/template.go`:

- Replace `execute(k, val, data)` with the low-level `Render(text, data)` operation.
- Use a constant internal Go template name, such as `template`, not an output key
  and never template contents.
- Preserve byte-to-string input conversion, `missingkey=error`, FuncMap, and the
  configured left/right delimiters.
- On parse failure, return a `Failure` declaring `template parsing failed` and
  retaining the parser error privately.
- On execution failure, use `WithContext(err, "template execution failed")`.
  If a cleaned ESO helper supplied public detail, it survives through Go's
  `ExecError` chain; otherwise the fixed stage is the complete public message.
- Return nil bytes on failure, even if the buffer contains partial output.
- Remove `errParse`/`errExecute` formats printing key/source/lower error text.

Example execution handling:

```go
if err := parsed.Execute(buf, stringValues); err != nil {
    return nil, errinfo.WithContext(err, "template execution failed")
}
```

Do not parse Go error strings to extract supposedly safe arguments or locations.
Do not add a general panic recovery layer around controllers. Existing Go template
function-panic conversion remains; built-in/type errors without a declaration get
only the stage message. Private causes retain detailed debugging information.

### 5.4 Separate output application from rendering

In this section, we extract "what to do with rendered bytes" from "how to execute a template",
in order to makes errors from that second processing stage follow the same public/private classification rules
defined in the previous sections.

Today, the template engine performs two different jobs:

1. Render: execute Go template text against fetched values.
2. Apply: interpret the resulting bytes and place them into an object.

In this section, we separate the two jobs.

```text
   Template text + fetched values
                |
              Render
                |
          Rendered bytes
                |
       ApplyValue / ApplyDocument
                |
      Modified in-memory object
```

NB: "Apply" here means modifying an in-memory object. It does not mean calling Kubernetes, using server-side apply, or persisting anything.

For that, we will move/rehome the non-engine-specific application code currently mixed into v2
`template.go` into `runtime/template/apply.go` and the relevant path helpers/tests as appropriate.

Proposed operations are:

```go
func ApplyValue(ref TemplateRef, rendered []byte, key, target string,
    obj client.Object, decoding esv1.ExternalSecretDecodingStrategy) error

func ApplyDocument(ref TemplateRef, rendered []byte, target string,
    obj client.Object, decoding esv1.ExternalSecretDecodingStrategy) ([]string, error)
```

#### ApplyValue

`ApplyValue` decodes and assigns one value to a caller-known key.

For example:

```yaml
data:
  connection-string: '{{ .username }}:{{ .password }}'
```

Rendering might produce:

```text
alice:example-password
```

At that point, the template engine’s execution job is finished.

ApplyValue receives:

- those rendered bytes;
- the destination key, connection-string;
- the target field, data;
- the destination Secret object;
- the configured decoding strategy;
- the configuration reference for error reporting.

It then:

1. Decodes the bytes if configured.

2. Assigns them to the selected destination:

```go
secret.Data["connection-string"] = rendered
```

The same operation can populate labels, annotations, or supported paths, using the existing application behavior.

It never executes the bytes as template text.

#### ApplyDocument

`ApplyDocument` parses the rendered YAML and applies its map entries or nested structure. Its
returned keys identify entries assigned to well-known map targets; they are
**sensitive internal bookkeeping**, never public identifiers. For nested structure
application, it can return nil keys. This small return value supports provenance
tracking for the existing generic-target second pass; it is not a new reporting
interface.

A KeysAndValues template might produce:

```yaml
username: alice
password: example-password
```

Here, the caller has not supplied one destination key. The output contains two keys.

ApplyDocument therefore:

1. Parses the rendered YAML.
2. For a map target such as data, processes and assigns its entries.
3. For a supported nested target, applies the parsed structure using the existing path-handling behavior.

For a Secret’s data map, the result would effectively be:

```go
secret.Data["username"] = []byte("alice")
secret.Data["password"] = []byte("example-password")
```

Again, this is an in-memory modification.

Worth noting: ApplyDocument returns []string to support tracking of which template produced each entry.

In the example, it returns:

```go
[]string{"username", "password"}
```

So that the caller can record:

```text
username → templateFrom[0].secret.items[0]
password → templateFrom[0].secret.items[0]
```

If those values are subsequently rendered again, the second pass can still identify the original configuration entry.

The returned keys are **not** public error context! A template could generate a password as a key:

```yaml
example-password: some-value
```
That is why the section calls the returned keys sensitive bookkeeping.

This return value is not fundamental to the current rendering.
It is simply the proposed mechanism for satisfying the provenance requirement.

#### Why does this matter?

**Successful rendering does not mean processing has finished successfully**. Which means it could leak errors into
events.

These bytes render successfully:

```yaml
password: [unfinished
```

But parsing them as YAML fails.

Other failures can occur during:

- decoding a rendered value;
- navigating a target path;
- converting the resulting structure into a typed Kubernetes object.

Those errors may contain sensitive rendered content. Consequently, the application functions must also create classified failures:

```go
   failure := errinfo.NewFailure(
       errinfo.Info{
           PublicMessage: "rendered YAML is invalid",
       },
       yamlError,
   )

   return errinfo.WithContext(failure, ref.SpecPath)
```

The public message identifies the responsible configuration and failing operation. The original YAML error remains a private cause.

This is the same error policy as rendering, not a separate mechanism.

#### Common to ApplyValue/ApplyDocument

These functions do not compile or execute template text. All their public errors
are a fixed operation plus the supplied structural `ref`. Private causes retain
parser/converter detail. Specifically:

| Existing failure point | Public declaration |
| --- | --- |
| `decoding.Decode` in either scope | `rendered value decoding failed` |
| `yaml.Unmarshal` into map or arbitrary nested data | `rendered YAML is invalid` |
| `parseTargetPath` | `invalid template target path` |
| `ToUnstructured`, `FromUnstructured`, incompatible field/root type, `setAtPath` | `rendered object application failed` |

Use `NewFailure` to declare those operation messages and `WithContext` to attach
the reference. Never append generated keys, raw target paths, or lower error text
to the public field. Detailed internal path-navigation errors may remain as causes.

Preserve all existing application semantics:

- case-insensitive well-known targets and case-sensitive nested paths;
- Secret byte/base64 handling versus non-Secret string handling;
- scope-specific decoding behavior, including existing nested-path behavior;
- map merging, metadata initialization, and the maximum array index;
- `tryParseYAML`'s intentional fallback to the original string;
- application order and precedence, without introducing unrelated sorting.

This is not an atomicity refactor. Earlier operations may mutate a working object
before a later one fails. Controllers must not persist that object or begin a
provider push after rendering/application fails.
At implementation time, we will need to care about not add an expensive deep-copy
layer per template.

### 5.5 Parser and controller migration

In `pkg/controllers/templating/parser.go`:

1. Replace `Exec template.ExecFunc` with `Render template.RenderFunc`.
2. Give the parser its known template-spec root (`spec.target.template` or
   `spec.template`) for building references. This is a caller-owned literal.
3. In `MergeTemplateFrom`, retain the entry index and pass its reference to
   `MergeConfigMap`, `MergeSecret`, and `MergeLiteral`.
4. In source-item loops retain the item index; construct `Template{Ref, Text}`
   after resolving the item. Keep source lookup and missing-item behavior, but
   classify missing-source/item failures with fixed public text.
5. Values: call Render, then ApplyValue with the existing `k.Key`.
6. KeysAndValues/literal: call Render, then ApplyDocument. No map whose key is
   the template source is needed anymore.
7. `MergeMap` receives a reference to the containing data/metadata field and loops
   over entries, rendering each value and applying it to its existing map key.
   Never use that key as public context. Reject unsupported template scopes with
   a fixed public message; do not turn the old invalid-scope error into a no-op
   when moving dispatch out of v2 Execute.
8. Classify actual `p.Client.Get` failures through the API classifier described
   below. Do not label fetch errors as template syntax errors.

Update `ApplyTemplate` in the ExternalSecret controller and `applyTemplate` in the
PushSecret controller to construct the new parser and pass the correct field
references. Preserve templateFrom/data/metadata precedence and merge policies.

Also migrate `cmd/esoctl/template.go`, a non-controller caller of EngineForVersion
and Parser. Set its parser root from the source object's supported kind
(ExternalSecret or PushSecret), replace `Exec` with `Render`, and update
`executeTemplate`'s MergeMap calls with the appropriate field references. Retain
its preloaded Secret/ConfigMap sources and successful output behavior. Filter errors
at the command's final error-return boundary too; YAML/source-file parsing errors
must not bypass the reporting policy. Successful CLI output intentionally contains
the rendered Secret and is not an error-reporting channel.

Migrate tests and callers before removing the old `ExecFunc` and map-based v2
`Execute`. Do not leave an alternate execution path that still publishes raw
errors. Characterization tests for the old successful behavior should accompany
this refactor so security changes do not silently change output.

### 5.6 Preserve provenance through generic-target double rendering

`renderTemplatedManifest` currently renders some sources into a temporary Secret
and then executes `tempSecret.Data` as templates.
We do not want to alter this behavior in this patch, as it would break our users.

Maintain an internal `map[string]TemplateRef` alongside the temporary data map:

- After a successful Values application to data, record its destination key's
  reference, even if the assigned bytes equal the previous value.
- After a successful KeysAndValues application to data, use ApplyDocument's
  returned keys to record the originating reference for each assigned entry.
- Later assignments overwrite the recorded reference just as they overwrite data.
- Labels/annotations and non-data paths do not create temporary data references.
- In the second pass, create `Template{Ref: dataRefs[key], Text: string(value)}`.
  Do not use `key` or `value` to invent a public reference.
- If an unexpected entry lacks provenance, use the containing templateFrom entry
  as a coarse fallback; never fall back to the data key.

This can be an optional data-reference map on the parser used by this collector;
ordinary Secret/PushSecret rendering need not maintain it. It is internal and
must not be serialized or logged.

Literal and inline generic templates use the same Render/application operations.
Remove the wrapper that prints `targetPath` after failed merged-template execution.
A second-pass parse failure must identify the original configuration source, not
print first-pass secret-derived output. Pass numbers are not required initially.

### 5.7 Rewrite paths remain explicit

Both rewrite paths execute Go templates directly and must adopt errinfo even
though they do not render Kubernetes objects:

- `runtime/esutils/utils.go:RewriteTransform`: classify parse/execute failures;
  remove public printing of the current key. Keep parse-once behavior, `.value`
  input, and existing delimiter/missing-key options.
- `RewriteRegexp`: declare `invalid rewrite regular expression`; keep the regexp
  parser error private.
- `RewriteMap`: replace a purely textual numeric-index wrapper with approved
  `WithContext` when the index should be visible. Enclosing controllers can add the
  known dataFrom entry reference without formatting fetched keys.
- PushSecret `compileRewrite`: classify regexp/parse/execute errors; preserve
  compilation outside the key loop.
- `rewriteWithKeyMapping`: replace `rewrite[%d] on key %q: %w` with approved
  index-only context. A previous transform can make `currentKey` secret-derived.

Do not route rewrites through the Kubernetes application functions, or silently
change their template options to match v2's main renderer.

## 6. Kubernetes classification and controller publication

### 6.1 Classify real client errors at their origins

Add `ClassifyAPIError(err error) error` in `pkg/controllers/util/apierr.go`.
Return nil for nil; otherwise construct `errinfo.NewFailure` with the original
error as cause and a fixed declaration selected from structured predicates:

| Predicate, checked in order | Public message |
| --- | --- |
| `apierrors.IsConflict` | `Kubernetes resource version conflict` |
| `IsNotFound` | `Kubernetes resource not found` |
| `IsAlreadyExists` | `Kubernetes resource already exists` |
| `IsInvalid` | `Kubernetes object validation failed` |
| `IsForbidden` / `IsUnauthorized` | `Kubernetes API access denied` |
| `IsTimeout` / `IsServerTimeout` | `Kubernetes API request timed out` |
| `IsTooManyRequests` | `Kubernetes API request throttled` |
| Other non-nil client error | `Kubernetes API request failed` |

Never copy `Status().Message`, arbitrary status causes/field paths, webhook denial
text, HTTP bodies, object dumps, or rejected values into the public declaration.
Only call this helper around actual Kubernetes client operations, not around a
combined template/provider path. It classifies display; it does not decide retries.

This closes the post-render case that removing `fail` cannot fix:

```yaml
target:
  template:
    engineVersion: v2
    metadata:
      labels:
        leak: '{{ printf "%s!" .password }}'
    data:
      password: '{{ .password }}'
```

Rendering succeeds, but `!` makes the label invalid. The API rejection can echo the
password. The Event and condition should instead contain:

```text
could not update secret: Kubernetes object validation failed
```

### 6.2 ExternalSecret origin migration

Replace broad `ctrlutil.Safe(err)` approval at these sites:

| Site | Change |
| --- | --- |
| `createSecret`: `r.Create` | Classify the actual API error. Leave mutation errors classified by their producers. |
| `updateSecret`: both `r.Update` paths | Keep conflict handling; classify other API failures. |
| `updateSecret`: missing target / immutable data guard | Construct fixed public information around the existing internal error/sentinel. |
| `applyOwnership` | Declare fixed ownership/reference failures, retaining sentinels and nested causes privately; omit owner metadata. |
| Get/List/Delete failures in cleanup, orphan deletion, target resolution | Classify actual API calls after existing NotFound handling. |
| Deletion policy refusing a non-Owner target | Declare the fixed policy requirement; preserve the current nil-error return after reporting. |
| Generic target get/create/update/delete | Classify API origins, not all errors returned by manifest rendering. |
| Generic target configuration checks | Declare the existing fixed requirements for enablement, apiVersion, and kind. |
| `GetProviderSecretData` failures | No blanket classification: retain supplied public information if present; unknown provider/data-processing errors use the operation fallback. |

Hash/conversion/key-validation errors outside the new rendering boundary remain
private unless their own producer deliberately supplies a declaration. Do not mark
them safe simply because their wrapper was written by ESO.

### 6.3 Compose one failure for Event, condition, and returned error

Evolve ExternalSecret `markAsFailed` to return the internally classified error:

```go
func (r *Reconciler) markAsFailed(
    msg string, err error, es *esv1.ExternalSecret,
    counter prometheus.Counter, reason string,
) error {
    reported := errinfo.WithContext(err, msg)
    message := errinfo.PublicMessage(reported, msg)
    r.recorder.Event(es, v1.EventTypeWarning, esv1.ReasonUpdateFailed, message)
    condition := NewExternalSecretCondition(esv1.ExternalSecretReady,
        v1.ConditionFalse, reason, message)
    SetExternalSecretCondition(es, *condition)
    counter.Inc()
    return reported
}
```

Existing fixed operation messages remain. Retrying callers return the classified
`reported` error; non-retryable callers ignore it and retain their nil-error
return. The final return gate projects it later. Do not add the same context a
second time at that gate.

This makes the Event and condition use the same string while allowing a returned
error to retain the same approved context and its original control-flow identity.

### 6.4 Return filtering and deferred status errors

ExternalSecret `Reconcile` already has named returns. Register a sanitizer as the
**first defer**, before metrics and status-update defers:

```go
defer func() {
    err = errinfo.PublicError(err, "ExternalSecret reconciliation failed")
}()
```

It runs last. The existing status defer can inspect the original main error,
handle status conflicts, and assign an API-classified status failure before the
projection happens. Do not pass `err` as an eagerly evaluated defer argument.
Do not return an already projected error from lower helpers and project it again.

Preserve these behaviors:

| Existing decision | Required result |
| --- | --- |
| Secret target write conflict | `Requeue: true`, nil error. |
| Generic-target write conflict | Existing one-second requeue, nil error. |
| Status-update conflict | Immediate requeue only when there is no main error. |
| Ownership / immutable non-retryable branches | Existing condition reason and nil returned error. |
| Provider/render failure | Non-nil returned error; existing backoff. |
| `esv1.NoSecretErr` | Existing deletion-policy behavior before publication. |
| PushSecret `locks.ErrConflict` | Existing immediate requeue with nil error. |
| SecretStore `errValidationUnknown` | Existing Ready=True reason and interval requeue. |

Recheck for framework-specific error markers before introducing projection to
additional controllers. The inspected paths do not use `TerminalError`; if that
changes, handle its behavior before removing its cause chain.

### 6.5 Logs are sinks too

A final return gate does not protect earlier logs. Pass a public projection to
explicit `log.Error` calls, and remove raw errors from Info fields on covered
paths. For example:

```go
log.Error(errinfo.PublicError(err, "unable to update status"),
    "unable to update status")
```

Cover ExternalSecret read/patch/cache errors, both status defers, generic-target
validation and informer errors, generator-state commit/rollback errors,
PushSecret Get/status Patch/lock-conflict logs, and SecretStore validation and
status-patch logs. API classification belongs at the actual API operation.

In `updateSecret`, replace debug key lists (`added`, `updated`, `removed`, `emptied`)
with counts: a generated key can itself be a password. Keep the key-diff algorithm;
change its logged representation and the misleading "keys are safe" comment.
Resource identity from controller configuration can remain, but do not introduce
rendered metadata or fetched keys as a substitute debug field.

No raw-cause debug toggle is added by this patch. Retaining causes permits internal
inspection; it is not permission to send them to ordinary debug logs. A future
restricted debugging facility would need an explicit access/retention policy.

### 6.6 Other controller sinks

| Controller / file | Required implementation |
| --- | --- |
| PushSecret `markAsFailed` | Accept `(base string, err error, ps, syncState)`, compose with `WithContext`, use one public string, and return the classified error for retrying callers. Preserve sync-state updates. |
| PushSecret callers | Replace `err.Error()` and error-interpolating strings with operation constants plus separate errors: store mapping, source resolution, provider write/delete/cleanup. Preserve partial-success state and lock handling. |
| PushSecret template failure | It currently returns without marking a condition. Keep that behavior; rendering classification and outward projection protect it without adding a new Event. |
| PushSecret `Reconcile` | Use named returns and a first-defer projection; sanitize its status-Patch log separately. |
| SecretStore `client_manager.go` | Replace provider-resolution `Safe` with a fixed declaration wrapping `ErrProviderResolution` and its cause. Do not approve `NewClient` error text. |
| SecretStore `validateStore` | Use one filtered/composed message for client/validation failure Event and condition. Unknown unclassified errors retain existing generic bases; explicit provider detail may be included. Preserve `ReasonProviderNotFound` and `errValidationUnknown` behavior. |
| Store and ClusterStore outer Reconcile functions | Project outward errors after common reconciliation has checked its sentinels; sanitize explicit common logs too. |
| ClusterPushSecret / ClusterExternalSecret `toNamespaceFailures` | Replace persisted raw `err.Error()` reasons with operation context plus approved detail; classify their actual API origins. Existing fixed markAsFailed messages can stay. |
| GeneratorState `markAsFailed` | Replace `%v` interpolation with approved context; unclassified cleanup failures stay generic. |
| Webhook configuration update failure | Replace raw error Eventf/log output with API classification and public projection. |

Review all remaining Event emitters and condition/failure-message builders before
claiming broad publication coverage. Do not add Events to controllers that
currently only publish conditions.

Normal Events also need review. ExternalSecret's missing-provider-secret message
can report only the actual `spec.data[i]` or `spec.dataFrom[i]` index rather than a
remote key. Preserve reasons/deletion behavior. Generic-target success Events use
kind/name from configuration, not rendered output; those identifiers can stay.

## 7. Migrating the existing safe-message mechanism

This is a replacement of representation and naming **within one mechanism**, not
a second safety policy operating alongside the old one.

1. Add `runtime/errinfo`, moving/adapting rune limiting and unwrap-aware selection
   tests from `pkg/controllers/util/statuserr_test.go` (see also previous section's
   for future proofing).
2. Introduce `Info`/`Failure` and explicit declarations at existing safe call sites.
   Remove broad API approval before relying on those details for Events.
3. Replace `SafeMessage` consumers with `PublicMessage` and context composition.
   If intermediate commits need `ctrlutil.SafeMessage`, make it delegate to the
   new extractor with an empty fallback; do not teach it to approve raw causes.
4. Migrate `Safe(err)` calls to explicit declarations or API classification. Do
   not implement a permanent compatibility helper that blindly copies
   `err.Error()` into `PublicMessage`.
5. Remove the old `safeError` implementation and unused delegates when callers
   are migrated. A search for `ctrlutil.Safe(` must find no remaining production
   or test usages at completion.
6. Update tests that previously expected the target name or arbitrary marked text
   to appear. Replace them with explicit-declaration tests, not weaker assertions.

The old innermost marker protected against approving composed strings. The new
representation does not derive public text from the cause at all. This is why it
can safely give an outer explicit declaration precedence, including withholding
inner detail with an empty message.

## 8. Provider adoption now and in the future

This section was AI generated, take that with a pinch of salt.

### 8.1 No new provider method or callback

`apis/externalsecrets/v1/provider.go` continues returning ordinary Go `error` values
from provider/client methods. Providers opt in by returning `*errinfo.Failure`.
No `SafeMessage()` method is added to provider interfaces, and controllers do not
ask an external system to explain an error after it occurred.

Example pattern, using a provider's existing structured SDK error classification:

```go
value, err := sdk.GetSecret(ctx, request)
if err != nil {
    // Preserve existing missing-secret translation and other control contracts.
    if isMissingSecret(err) {
        return nil, esv1.NoSecretErr
    }

    message := "provider request failed"
    switch {
    case isAccessDenied(err):
        message = "provider access denied"
    case isThrottled(err):
        message = "provider request throttled"
    }
    return nil, errinfo.NewFailure(errinfo.Info{PublicMessage: message}, err)
}
return value, nil
```

`isMissingSecret`/`isAccessDenied`/`isThrottled` above stand for each SDK's structured
predicates/codes, not message substring matching. AWS Secrets Manager's existing
`smithy.APIError.ErrorCode()` handling is one concrete adoption point. Keep metrics
observation and existing missing-secret translation in their current positions.

### 8.2 Provider author rules

- Declare fixed messages or reviewed fields that contain no secret-derived data.
- Never put an SDK `Error()`, response body, authentication URL, token, property
  value, or fetched key into `Info.PublicMessage` without a deliberate safety
  review. A public declaration that leaks is a clear producer implementation bug.
- Keep the original cause for `errors.As`/`errors.Is`. Do not project with
  `PublicError` inside the provider: the controller may still need SDK behavior.
- Preserve `NoSecretErr`, `NotModifiedErr`, existing sentinels, and any direct
  equality assumptions in that provider's tests. Do not indiscriminately wrap all
  success/control sentinels during adoption.
- A provider may intentionally return `NewFailure(Info{}, cause)` to withhold
  public detail, even if a lower layer declared something more specific.
- Successful secret data remains sensitive by default. `Info.PublicMessage` does
  not label the accompanying payload, source keys, or other response fields public.
- Returned failures and direct provider logging are separate responsibilities;
  review local log calls as part of adoption rather than assuming wrapping fixes
  logs already emitted by an SDK.

### 8.3 Adoption checklist for each provider module

1. Import `github.com/external-secrets/external-secrets/runtime/errinfo`. Many
   providers already depend on the runtime module; use the repository's existing
   multi-module dependency workflow where wiring is needed.
2. Identify structured failure categories at SDK/auth/parser boundaries. Start
   with client creation, Get, map/list retrieval, Push, Delete, Exists, Validate,
   and Close paths that actually return errors.
3. Declare only reviewed public messages and retain private causes. Do not
   implement the migration as a global textual replacement of every `return err`.
4. Keep behavior and result semantics unchanged. Add tests proving sentinel/SDK
   identity survives wrapping.
5. Inject synthetic credentials/payloads into raw SDK errors. Assert public
   extraction and Error() exclude them while internal cause inspection retains
   the original error.
6. Verify unclassified failures still produce a useful controller fallback.
7. Update provider-specific troubleshooting guidance if users formerly depended
   on raw provider error text.

The security fix does not wait for this rollout: unknown provider failures already
fall back safely at publication. Subsequent adoption improves troubleshooting.

### 8.4 Future ESO/provider architectures

For a future process boundary, represent the public `Info` as response metadata.
The local adapter reconstructs `Failure` with that metadata and whatever private
local error representation the protocol allows. Go error pointers and SDK types
do not cross the process boundary, and this plan does not pretend they do.

No extra RPC is needed to obtain a public message. Treat the declaration as part
of the provider contract; protect any separately transported private detail from
Events/status/logs. Unknown older responses use a generic fallback.

Behavior metadata can be added later if controllers actually consume it. For
example, retry guidance would need an explicit unspecified state, not a bool whose
zero value accidentally means "never retry". Agree on controller precedence and
wire compatibility before adding it. It must not silently replace Kubernetes
conflict checks or secret-not-found deletion semantics. No such field/interface
is introduced by this security fix.

## 9. Tests that demonstrate the design and close the leak

This section was AI generated, take that with a pinch of salt.

### 9.1 `runtime/errinfo` contract tests

Use a distinctive fake value such as `ESO_EVENT_CANARY_7F92` in private causes.
Test:

- constructor always creates a failure; nil cause is valid;
- Error() uses only its own message, with basic fallback for empty Info;
- raw errors select fallback, never their Error() text;
- ordinary `%w` wrappers carrying a canary reveal only the enclosed public text;
- outer declaration overrides inner declaration;
- empty outer declaration selects fallback instead of inner detail;
- joined errors follow first-matching Failure semantics, including an empty match;
- nil handling for PublicMessage, WithContext, and PublicError;
- WithContext combines only approved text and preserves the original error chain;
- `errors.Is`/`errors.As` retain SDK/sentinel identities internally;
- PublicError has no unwrap chain and no SDK identity;
- rune-aware bounds work with multibyte text and long declared messages;
- filtering does not call an unknown error's Error() (use an error implementation
  whose Error method panics to make accidental formatting visible).

Test raw causes are retained rather than asserting they were destroyed. Negative
formatting assertions belong on public strings/projections, not reflective dumps
of an internal Failure that intentionally contains a private cause.

### 9.2 Renderer and application tests

Characterize successful output before splitting v2 Execute. Reuse existing tests
for delimiters, missing keys, decoding, labels/annotations, Secret encoding,
nested targets, merge precedence, and invalid paths. Move tests with the code
rather than preserving a second map-based execution API solely for testing.

Regression matrix:

| Input | Required public result |
| --- | --- |
| Removed `{{ fail (printf "%v" .) }}` | Parsing failure; no source or fetched values. |
| `{{ mustToDate "2006-01-02" .password }}` | Execution stage plus reviewed helper detail if classified. |
| `{{ mustRegexMatch (printf "[%s" .password) "" }}` | Execution stage; no pattern text. |
| Non-must `regexFind` with that invalid pattern | Classified function-panic failure, not raw panic text. |
| `{{ slice .password -1 }}` or missing input | Generic execution failure for unclassified Go built-in/evaluator errors. |
| Secret-sourced text with canary and unmatched `{{` | Parsing failure identifying the source spec entry, not its contents. |
| Rendered malformed YAML containing canary | YAML application failure; no document excerpt. |
| Generated canary key with invalid Base64 value | Decode failure; no generated key. |
| Invalid typed field conversion containing canary | Object application failure; no rejected value. |
| First generic pass outputs malformed template text | Second-pass failure retaining first source reference. |
| Rewrite uses canary as current/transformed key | Numeric spec context only; no key in public output. |

Add direct helper tests as well: engine-level filtering must not hide a helper
regression. Preserve existing `decrypt_test.go` sentinel checks. Replace old engine
assertions about raw Go error strings with exact expected public messages and
checks that private causes remain inspectable.

Use fake Secret/ConfigMap source objects in `parser_test.go`; verify references
track actual list indexes and inline fields omit keys. Test provenance when two
sources overwrite the same temporary key, including equal-value overwrites.
Assert application failure prevents Kubernetes write/provider push, not that every
in-memory intermediate object is rolled back. Do not make unrelated provider
partial-success behavior atomic. Add `cmd/esoctl/template_test.go` coverage for both
supported source kinds, retained successful CLI output, and sanitized error returns
from malformed template/source data; isolate and restore its package-global flags.

### 9.3 API and controller publication tests

Construct real-shaped rejected-value errors:

```go
raw := apierrors.NewInvalid(
    schema.GroupKind{Kind: "Secret"}, "output",
    field.ErrorList{
        field.Invalid(field.NewPath("metadata", "labels").Key("leak"),
            canary+"!", "invalid label value"),
    },
)
classified := ctrlutil.ClassifyAPIError(raw)
reported := errinfo.WithContext(classified, "could not update secret")
```

Assert the exact public message from section 6.1, continued internal API identity,
and absence of raw StatusError/cause identity after projection. Cover Forbidden
and unknown webhook-style StatusErrors containing canaries, plus every predicate
in the classifier table.

Extend `pkg/controllers/externalsecret/markasfailed_test.go` to return its existing
buffered `*record.FakeRecorder` from the fixture. Read the recorded Event without
sleeping and assert both surfaces:

```go
select {
case got := <-recorder.Events:
    require.Equal(t, "Warning UpdateFailed "+wantMessage, got)
default:
    t.Fatal("expected a failure event")
}
require.Equal(t, wantMessage, readyCondition(t, es).Message)
```

Also assert condition status/reason, metric increment, and the returned classified
error. Cover ordinary malicious wrappers, explicit outer declarations, empty
outer declarations, and unknown provider errors. Replace old `Safe` tests; do not
reintroduce arbitrary text approval just to keep their assertions passing.

Add corresponding PushSecret tests for preserved sync state, SecretStore tests for
all validation-result branches, namespace-failure status tests, and GeneratorState
cleanup condition tests. Reuse `client_manager_saferr_test.go` provider registration
fixtures carefully; do not parallelize tests mutating global registrations.

### 9.4 Integration and control-flow coverage

Use the existing ExternalSecret envtest suite for the invalid-label scenario.
Provide a valid data/store reference and fake provider returning the canary. Wait
for a matching Warning Event and Ready=False; filter Events by involved-object UID.
Assert no new target was stored. Repeat for an existing target and confirm the
rejected write did not change persisted data/metadata.

A fake client does not perform real API validation. In unit tests, explicitly
inject a `NewInvalid` error via a client wrapper/interceptor. Use envtest for actual
rejection. Do not run a second reconciler over the existing manager's test objects;
use isolated fixtures for manual Reconcile/result tests.

Test every branch in the retry table, especially a status-update conflict versus
a non-conflict status error assigned by a defer. Assert results after all defers.
Capture explicit logs with a test logr sink and verify public projections do not
contain canary plaintext, base64 forms, or selected substrings. Pair absence tests
with exact approved output: checking only absence of a full password misses
partial leaks. Do not present finite tests as proof against behavioral side channels.

## 10. Implementation order, documentation, and validation

This section was AI generated, take that with a pinch of salt.

Deliver focused, test-backed changes in this order:

1. Add/evolve errinfo and its complete selection/context/projection tests.
2. Migrate existing Safe origins and controller sinks, including API classification
   and outward/log filtering. This closes the publication gap for unknown errors
   without waiting for every provider or the rendering refactor.
3. Remove `fail`; split rendering/application; carry TemplateRef through parser,
   generic-target double rendering, and rewrite paths. Clean helper declarations
   and update success-behavior characterization tests alongside the refactor.
4. Complete cross-controller and envtest regressions, documentation, and the
   provider adoption guide. Provider-wide enrichment can follow incrementally;
   generic fallbacks must already protect unmodified providers.

Do not describe `fail` removal alone, helper cleanup alone, or status-only filtering
as the completed security fix. There is one publication policy across these steps.

Update:

- `runtime/template/v2/sprig/README.md`: `fail` exclusion and maintained divergence;
- `docs/guides/templating.md`: unavailable function, new source references, reduced
  unsafe detail, and unchanged successful rendering semantics;
- `docs/guides/security-best-practices.md`: public-diagnostics boundary and template
  author/output restrictions that remain necessary;
- provider contribution guidance: declaring public Info, preserving causes and
  control contracts, and testing adoption;
- release notes: `fail` compatibility break, public message changes, and loss of
  raw parser/SDK/API detail from public channels.

Do not recommend debugging with real secrets or enabling unfiltered cause logging.
Failures are still observable through reasons, stages, structural references,
metrics, and reviewed producer messages.

Review checkpoints (not automatic proofs):

```sh
rg -n 'ctrlutil.Safe\(|SafeMessage\(' pkg runtime providers --glob '*.go'
rg -n 'Eventf?\(|New.*Condition\(|Reason:.*Error' pkg/controllers --glob '*.go'
rg -n 'err.Error\(\)|log.Error\(|Log.Error\(|"error",' pkg/controllers --glob '*.go'
rg -n 'EngineForVersion|ExecFunc|templating.Parser' runtime pkg cmd --glob '*.go'
rg -n 'template.New|tpl.New|\.Execute\(' runtime pkg/controllers cmd --glob '*.go'
```

Each remaining raw formatter needs an explicit explanation of whether it is
private error construction or a missed publication sink. Review public message
construction too: a declared string can still be wrong if its producer inserts
secret material. No generic key-name heuristic makes it safe.

Use repository Makefile targets, not direct `go test`, lint, or Helm commands:

- `make reviewable` for the review gate;
- `make test && make check-diff` before the implementation PR is ready;
- `make check-diff` again after all intended changes are committed, because the
  target requires a clean worktree and includes additional generation.

No generated CRD/API changes are expected. Inspect any generated diff and keep
unrelated changes out. Record unavailable tools/environment blockers explicitly.
These are implementation acceptance steps, not checks claimed for this
proposal-only document.

## 12. Definition of done

This section was AI generated, take that with a pinch of salt.

- One evolved error mechanism: Info/Failure, explicit producer declarations,
  context composition, and default-private public selection.
- Failure.Error is simple and safe; standard Unwrap preserves internal identity;
  outward projections carry no private cause.
- The first-Failure and empty-declaration rules are tested and documented.
- Events and matching conditions use the same composed public message.
- Arbitrary API/provider errors are never approved wholesale or used as fallback.
- `fail` is absent from all shared function maps.
- Render execution is separate from output application, with useful structural
  references and preserved successful behavior.
- Rewrites, Secret-sourced templates, generated keys, and generic double rendering
  cannot inject their contents into public error text.
- Post-render API rejections, covered explicit logs, and deferred return paths
  follow the same policy without changing retries or deletion/ownership semantics.
- Unmodified providers are protected by fallback; adopting providers can add useful
  public information without a new interface or provider callback.
- Remaining side channels, output capabilities, provider-internal logging, and
  non-diagnostic state are explicitly outside the completed claim.
- Tests, documentation, and repository validation gates pass before implementation
  is presented as ready.
