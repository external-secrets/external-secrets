## Problem Statement

What is the problem you're trying to solve?

## Related Issue

Fixes #...

## Proposed Changes

How do you like to solve the issue and why?

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
design(scope): improve design
```

Where `scope` is _optionally_ one of:
- charts
- release
- testing
- security
- templating

## AI Assistance disclosure

Did you use an script, LLM, or AI assisted development tool for this contribution?

AI assistance used: Yes / No

If yes provide details:

Tool(s) used:

Purpose of assistance:

Parts of the contribution affected:

Human validation performed:

## Review process

Review runs in two stages: automated review first, then a maintainer. If an automated reviewer
raises findings, this pull request stays with you until they are resolved, and a comment will
appear explaining what is outstanding. A `review/*` label tracks where it sits. See
[how review is routed](https://external-secrets.io/latest/contributing/process/#how-review-is-routed).

## Checklist

- [ ] I have read the [contribution guidelines](https://external-secrets.io/latest/contributing/process/#submitting-a-pull-request)
- [ ] All commits are signed with `git commit --signoff`
- [ ] My changes have reasonable test coverage
- [ ] All tests pass with `make test`
- [ ] I ensured my PR is ready for review with `make reviewable`
- [ ] I confirm that I understand the submitted changes and can explain them without relying on an AI tool.
- [ ] I have tested my changes on a live environment to confirm they are working
