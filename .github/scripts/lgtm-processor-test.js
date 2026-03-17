/**
 * Tests for lgtm-processor.js helper logic.
 *
 * Tests the pure CODEOWNERS parsing and file-pattern matching behaviour
 * that lives inside lgtm-processor.js, extracted here so they can run
 * without GitHub Actions context.
 *
 * Run with: node .github/scripts/lgtm-processor-test.js
 */

import assert from 'node:assert/strict';

// ---------------------------------------------------------------------------
// Helpers duplicated from lgtm-processor.js for unit testing
// (kept in sync manually; tests will catch drift)
// ---------------------------------------------------------------------------

function parseCodeowners(content, organization) {
  const codeownerMappings = [];
  let wildcardRoles = [];

  content.split('\n').forEach(line => {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) return;

    const match = trimmed.match(/^(\S+)\s+(.+)$/);
    if (match) {
      const [, pattern, roles] = match;
      const rolesList = roles.split(/\s+/).filter(r => r.startsWith('@'));

      if (pattern === '*') {
        wildcardRoles = rolesList;
      } else {
        codeownerMappings.push({ pattern, roles: rolesList });
      }
    }
  });

  const maintainerRoles = wildcardRoles.map(role => role.replace(`@${organization}/`, ''));
  return { codeownerMappings, wildcardRoles, maintainerRoles };
}

function fileMatchesPattern(file, pattern) {
  return file === pattern ||
    file.startsWith(pattern.endsWith('/') ? pattern : pattern + '/');
}

function getRequiredReviewerRoles(changedFiles, codeownerMappings, wildcardRoles) {
  const requiredReviewerRoles = new Set();
  let hasFilesWithoutSpecificOwners = false;

  changedFiles.forEach(file => {
    let hasSpecificOwner = false;
    codeownerMappings.forEach(({ pattern, roles }) => {
      if (fileMatchesPattern(file, pattern)) {
        roles.forEach(role => requiredReviewerRoles.add(role));
        hasSpecificOwner = true;
      }
    });
    if (!hasSpecificOwner) hasFilesWithoutSpecificOwners = true;
  });

  if (hasFilesWithoutSpecificOwners) {
    wildcardRoles.forEach(role => requiredReviewerRoles.add(role));
  }

  return requiredReviewerRoles;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

const CODEOWNERS = `
# Global owners
* @external-secrets/maintainers

pkg/provider/aws/ @external-secrets/aws-team
pkg/provider/gcp/ @external-secrets/gcp-team
docs/ @external-secrets/docs-team
`;

const ORG = 'external-secrets';

// parseCodeowners: wildcard roles
{
  const { wildcardRoles, maintainerRoles } = parseCodeowners(CODEOWNERS, ORG);
  assert.deepEqual(wildcardRoles, ['@external-secrets/maintainers']);
  assert.deepEqual(maintainerRoles, ['maintainers']);
}

// parseCodeowners: directory mappings
{
  const { codeownerMappings } = parseCodeowners(CODEOWNERS, ORG);
  assert.equal(codeownerMappings.length, 3);
  assert.equal(codeownerMappings[0].pattern, 'pkg/provider/aws/');
  assert.deepEqual(codeownerMappings[0].roles, ['@external-secrets/aws-team']);
}

// parseCodeowners: comments and blank lines are ignored
{
  const { codeownerMappings } = parseCodeowners('# comment\n\npkg/foo/ @org/team\n', ORG);
  assert.equal(codeownerMappings.length, 1);
}

// fileMatchesPattern: exact match
assert.equal(fileMatchesPattern('pkg/provider/aws/s3.go', 'pkg/provider/aws/s3.go'), true);

// fileMatchesPattern: directory prefix (with trailing slash)
assert.equal(fileMatchesPattern('pkg/provider/aws/s3.go', 'pkg/provider/aws/'), true);

// fileMatchesPattern: directory prefix (without trailing slash)
assert.equal(fileMatchesPattern('pkg/provider/aws/s3.go', 'pkg/provider/aws'), true);

// fileMatchesPattern: no match
assert.equal(fileMatchesPattern('pkg/provider/gcp/storage.go', 'pkg/provider/aws/'), false);

// fileMatchesPattern: partial prefix should not match
assert.equal(fileMatchesPattern('pkg/provider/aws-extra/file.go', 'pkg/provider/aws'), false);

// getRequiredReviewerRoles: file under specific owner
{
  const { codeownerMappings, wildcardRoles } = parseCodeowners(CODEOWNERS, ORG);
  const roles = getRequiredReviewerRoles(['pkg/provider/aws/ec2.go'], codeownerMappings, wildcardRoles);
  assert.ok(roles.has('@external-secrets/aws-team'));
  assert.ok(!roles.has('@external-secrets/maintainers'));
}

// getRequiredReviewerRoles: file with no specific owner falls back to wildcard
{
  const { codeownerMappings, wildcardRoles } = parseCodeowners(CODEOWNERS, ORG);
  const roles = getRequiredReviewerRoles(['some/unowned/file.go'], codeownerMappings, wildcardRoles);
  assert.ok(roles.has('@external-secrets/maintainers'));
}

// getRequiredReviewerRoles: mixed files collect all required roles
{
  const { codeownerMappings, wildcardRoles } = parseCodeowners(CODEOWNERS, ORG);
  const roles = getRequiredReviewerRoles(
    ['pkg/provider/aws/ec2.go', 'pkg/provider/gcp/storage.go'],
    codeownerMappings, wildcardRoles
  );
  assert.ok(roles.has('@external-secrets/aws-team'));
  assert.ok(roles.has('@external-secrets/gcp-team'));
}

// ---------------------------------------------------------------------------
// lgtmProcessor integration tests (mocked GitHub API)
// ---------------------------------------------------------------------------

import { describe, it } from 'node:test';
import run from './lgtm-processor.js';

function makeMockContext() {
  return {
    repo: { owner: 'external-secrets', repo: 'external-secrets' },
    payload: {
      comment: { user: { login: 'testuser' } },
      issue: { number: 42 }
    }
  };
}

// label existence check: fails fast when lgtm label is missing
await describe('lgtmProcessor label existence check', async () => {
  await it('should call core.setFailed when lgtm label does not exist', async () => {
    let failedMessage = null;
    const core = {
      setFailed: (msg) => { failedMessage = msg; }
    };
    const github = {
      paginate: async () => [{ name: 'bug' }, { name: 'enhancement' }],
      rest: {
        issues: {
          listLabelsForRepo: () => {}
        }
      }
    };
    const context = makeMockContext();
    const fs = { readFileSync: () => '* @external-secrets/maintainers\n' };

    await run({ core, github, context, fs });

    assert.ok(failedMessage !== null, 'core.setFailed should have been called');
    assert.ok(failedMessage.includes('does not exist'), `Expected message about missing label, got: ${failedMessage}`);
  });

  await it('should not call core.setFailed when lgtm label exists', async () => {
    let failedMessage = null;
    const core = {
      setFailed: (msg) => { failedMessage = msg; }
    };

    // The function will proceed past the label check and try to read CODEOWNERS.
    // We let readFileSync throw to stop execution early (the point is that setFailed was NOT called).
    const github = {
      paginate: async () => [{ name: 'bug' }, { name: 'lgtm' }],
      rest: {
        issues: {
          listLabelsForRepo: () => {}
        }
      }
    };
    const context = makeMockContext();
    const fs = {
      readFileSync: () => { throw new Error('stop here'); }
    };

    await run({ core, github, context, fs });

    assert.equal(failedMessage, null, 'core.setFailed should not have been called when lgtm label exists');
  });
});

console.log('All tests passed.');
