## Problem Statement

What is the problem you're trying to solve?

## Related Issue

Fixes #...

## Proposed Changes

How do you like to solve the issue and why?

## Breaking Changes / Upgrade Notes

> If this PR contains breaking changes or requires special upgrade steps, describe them here.
> Add the `breaking-change` label to ensure it appears prominently in the release notes.

N/A

## Release Note
<!--
Write your release note for deployers:
1. Enter your release note in the below block.
2. If no CHANGELOG release note is required, just write "NONE" within the block.
3. Focus on the impact and what the user/deployers needs to know.
Com
-->
```release-note

```

## Format

Please ensure that your PR follows the following format for the title:
```
feat(scope): add new feature
fix(scope): fix bug
docs(scope): update documentation
chore(scope): update build tool or dependencies
ref(scope): refactor code
clean(scope): provider cleanup
test(scope): add tests
perf(scope): improve performance
desig(scope): improve design
```

Where `scope` is _optionally_ one of:
- charts
- release
- testing
- security
- templating

## Checklist

- [ ] I have read the [contribution guidelines](https://external-secrets.io/latest/contributing/process/#submitting-a-pull-request)
- [ ] If this PR has breaking changes, I added the `breaking-change` label and described them in the section above
- [ ] All commits are signed with `git commit --signoff`
- [ ] My changes have reasonable test coverage
- [ ] All tests pass with `make test`
- [ ] I ensured my PR is ready for review with `make reviewable`
