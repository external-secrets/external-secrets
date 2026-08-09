# OpenBao Provider Development

## API

When implementing new functionality, please check the Vault provider first. If
the same functionality is implemented in the Vault provider: Use the same API if
there is no good reason to deviate.

## Testing Strategy

Much of the provider is tested using recorded HTTP traffic from interactions
with a real OpenBao server (stored in `testdata/http/<TestName>.yaml`). This
gives us very realistic tests, which still run in a few milliseconds.

### Re-recording Traffic

To re-record the traffic, run:

```bash
ESO_PROVIDER_OPENBAO_RERECORD=true go test .
```

This will:

- delete all previous recordings
- start an OpenBao Dev Server (requires `bao` to be on your `PATH`)
- run the tests while proxying all requests to OpenBao
- store the HTTP traffic
- stop the Dev Server

Before storing the HTTP traffic some cleanup is applied (see `getRecorder`),
this replaces values that are random (e.g. OpenBao "mount accessors") or
timestamp based (e.g. creation timestamps) with predictable values. While this
is not technically necessary and adds some complexity to the tests, it greatly
improves the readability of the `git diff`. When you have to rerecord the
traffic, it is recommended to:

1. run with `ESO_PROVIDER_OPENBAO_RERECORD=true`
1. `git add` the recordings
1. run again with `ESO_PROVIDER_OPENBAO_RERECORD=true`, if there are changes in
   the recording files, tweak the cleanup logic and go to step 1.
1. run again without `ESO_PROVIDER_OPENBAO_RERECORD=true` and see if the tests
   still pass.

### Limits of Traffic Recording/Replay

While this strategy is good for testing the CRUD operations on secrets, it will
not work very well for many Authentication methods, which often work with random
strings and might even have explicit replay protections, therefore:

- We will only apply the HTTP recording based tests to the UserPass auth method,
  which is rather static.
- For other auth methods, we will only validate that the OpenBao client has been
  configured as expected.

With this setup, we can still run extensive tests on our logic (e.g. token
caching) against the UserPass auth method and relying on the OpenBao client to
call whatever authentication method we have configured.
