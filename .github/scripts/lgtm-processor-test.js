'use strict';

const { test, describe } = require('node:test');
const assert = require('node:assert/strict');

const {
  parseCodeowners,
  fileMatchesPattern,
  getRequiredReviewerRoles,
  isRoleCovered,
  findUncoveredRoles,
  buildConfirmationMessage,
} = require('./lgtm-processor.js');

// ---------------------------------------------------------------------------
// parseCodeowners
// ---------------------------------------------------------------------------

describe('parseCodeowners', () => {
  test('parses wildcard and specific patterns', () => {
    const content = `
# Comment line
*  @external-secrets/maintainers @external-secrets/approvers
pkg/provider/aws/  @external-secrets/aws-reviewers
pkg/provider/gcp/  @external-secrets/gcp-reviewers
`;
    const { codeownerMappings, wildcardRoles, maintainerRoles } = parseCodeowners(
      content,
      'external-secrets'
    );

    assert.deepEqual(wildcardRoles, [
      '@external-secrets/maintainers',
      '@external-secrets/approvers',
    ]);
    assert.deepEqual(maintainerRoles, ['maintainers', 'approvers']);
    assert.equal(codeownerMappings.length, 2);
    assert.deepEqual(codeownerMappings[0], {
      pattern: 'pkg/provider/aws/',
      roles: ['@external-secrets/aws-reviewers'],
    });
    assert.deepEqual(codeownerMappings[1], {
      pattern: 'pkg/provider/gcp/',
      roles: ['@external-secrets/gcp-reviewers'],
    });
  });

  test('ignores comment lines and blank lines', () => {
    const content = `# Full comment\n\n# Another comment\n`;
    const { codeownerMappings, wildcardRoles } = parseCodeowners(content, 'org');
    assert.equal(codeownerMappings.length, 0);
    assert.deepEqual(wildcardRoles, []);
  });

  test('handles file with only a wildcard entry', () => {
    const content = `*  @org/team-a\n`;
    const { codeownerMappings, wildcardRoles, maintainerRoles } = parseCodeowners(
      content,
      'org'
    );
    assert.deepEqual(wildcardRoles, ['@org/team-a']);
    assert.deepEqual(maintainerRoles, ['team-a']);
    assert.equal(codeownerMappings.length, 0);
  });

  test('ignores tokens that do not start with @', () => {
    const content = `pkg/  @org/team-a some-non-role\n`;
    const { codeownerMappings } = parseCodeowners(content, 'org');
    assert.deepEqual(codeownerMappings[0].roles, ['@org/team-a']);
  });

  test('accumulates roles from multiple wildcard lines', () => {
    const content = `*  @org/team-a\n*  @org/team-b\n`;
    const { wildcardRoles } = parseCodeowners(content, 'org');
    assert.deepEqual(wildcardRoles, ['@org/team-a', '@org/team-b']);
  });

  test('deduplicates roles across multiple wildcard lines', () => {
    const content = `*  @org/team-a @org/team-b\n*  @org/team-b @org/team-c\n`;
    const { wildcardRoles } = parseCodeowners(content, 'org');
    assert.deepEqual(wildcardRoles, ['@org/team-a', '@org/team-b', '@org/team-c']);
  });
});

// ---------------------------------------------------------------------------
// fileMatchesPattern
// ---------------------------------------------------------------------------

describe('fileMatchesPattern', () => {
  test('exact file match', () => {
    assert.equal(fileMatchesPattern('README.md', 'README.md'), true);
  });

  test('file inside a directory pattern (with trailing slash)', () => {
    assert.equal(fileMatchesPattern('pkg/provider/aws/secret.go', 'pkg/provider/aws/'), true);
  });

  test('file inside a directory pattern (without trailing slash)', () => {
    assert.equal(fileMatchesPattern('pkg/provider/aws/secret.go', 'pkg/provider/aws'), true);
  });

  test('file in a sibling directory does not match', () => {
    assert.equal(fileMatchesPattern('pkg/provider/gcp/secret.go', 'pkg/provider/aws/'), false);
  });

  test('partial prefix does not match as directory', () => {
    // 'pkg/provider/aws_extra/file.go' should NOT match 'pkg/provider/aws'
    // because 'pkg/provider/aws/' is not a prefix of 'pkg/provider/aws_extra/...'
    assert.equal(
      fileMatchesPattern('pkg/provider/aws_extra/file.go', 'pkg/provider/aws'),
      false
    );
  });

  test('nested subdirectory matches parent pattern', () => {
    assert.equal(
      fileMatchesPattern('pkg/provider/aws/secrets/store.go', 'pkg/provider/aws/'),
      true
    );
  });

  test('root-anchored pattern with leading slash does not match without normalization', () => {
    // Leading slash patterns are not normalized, so they won't match plain paths
    assert.equal(fileMatchesPattern('pkg/file.go', '/pkg'), false);
  });
});

// ---------------------------------------------------------------------------
// getRequiredReviewerRoles
// ---------------------------------------------------------------------------

describe('getRequiredReviewerRoles', () => {
  const codeownerMappings = [
    { pattern: 'pkg/provider/aws/', roles: ['@org/aws-reviewers'] },
    { pattern: 'pkg/provider/gcp/', roles: ['@org/gcp-reviewers'] },
  ];
  const wildcardRoles = ['@org/maintainers'];

  test('returns roles for matched file', () => {
    const { requiredReviewerRoles, hasFilesWithoutSpecificOwners } = getRequiredReviewerRoles(
      ['pkg/provider/aws/file.go'],
      codeownerMappings,
      wildcardRoles
    );
    assert.deepEqual([...requiredReviewerRoles], ['@org/aws-reviewers']);
    assert.equal(hasFilesWithoutSpecificOwners, false);
  });

  test('falls back to wildcard roles for unmatched file', () => {
    const { requiredReviewerRoles, hasFilesWithoutSpecificOwners } = getRequiredReviewerRoles(
      ['docs/readme.md'],
      codeownerMappings,
      wildcardRoles
    );
    assert.equal(requiredReviewerRoles.has('@org/maintainers'), true);
    assert.equal(hasFilesWithoutSpecificOwners, true);
  });

  test('combines roles for multiple matched files from different patterns', () => {
    const { requiredReviewerRoles } = getRequiredReviewerRoles(
      ['pkg/provider/aws/file.go', 'pkg/provider/gcp/file.go'],
      codeownerMappings,
      wildcardRoles
    );
    assert.equal(requiredReviewerRoles.has('@org/aws-reviewers'), true);
    assert.equal(requiredReviewerRoles.has('@org/gcp-reviewers'), true);
  });

  test('adds wildcard roles when some files have no specific owner', () => {
    const { requiredReviewerRoles } = getRequiredReviewerRoles(
      ['pkg/provider/aws/file.go', 'unowned/file.go'],
      codeownerMappings,
      wildcardRoles
    );
    assert.equal(requiredReviewerRoles.has('@org/aws-reviewers'), true);
    assert.equal(requiredReviewerRoles.has('@org/maintainers'), true);
  });

  test('empty file list yields empty roles', () => {
    const { requiredReviewerRoles } = getRequiredReviewerRoles([], codeownerMappings, wildcardRoles);
    assert.equal(requiredReviewerRoles.size, 0);
  });
});

// ---------------------------------------------------------------------------
// isRoleCovered
// ---------------------------------------------------------------------------

describe('isRoleCovered', () => {
  const codeownerMappings = [
    { pattern: 'pkg/', roles: ['@org/pkg-reviewers'] },
    { pattern: 'pkg/provider/', roles: ['@org/provider-reviewers'] },
    { pattern: 'pkg/provider/aws/', roles: ['@org/aws-reviewers'] },
  ];

  test('role directly present in commenter roles is covered', () => {
    const commenterRoles = new Set(['@org/aws-reviewers']);
    assert.equal(isRoleCovered('@org/aws-reviewers', commenterRoles, codeownerMappings), true);
  });

  test('broader commenter role covers a more specific required role', () => {
    // commenter has pkg/ role, required is pkg/provider/aws/ role
    const commenterRoles = new Set(['@org/pkg-reviewers']);
    assert.equal(isRoleCovered('@org/aws-reviewers', commenterRoles, codeownerMappings), true);
  });

  test('parent role covers intermediate child role', () => {
    const commenterRoles = new Set(['@org/pkg-reviewers']);
    assert.equal(isRoleCovered('@org/provider-reviewers', commenterRoles, codeownerMappings), true);
  });

  test('sibling role does not cover another sibling', () => {
    const codeownerMappingsLocal = [
      { pattern: 'pkg/provider/aws/', roles: ['@org/aws-reviewers'] },
      { pattern: 'pkg/provider/gcp/', roles: ['@org/gcp-reviewers'] },
    ];
    const commenterRoles = new Set(['@org/aws-reviewers']);
    assert.equal(
      isRoleCovered('@org/gcp-reviewers', commenterRoles, codeownerMappingsLocal),
      false
    );
  });

  test('role not present and no hierarchy coverage returns false', () => {
    const commenterRoles = new Set(['@org/unrelated-team']);
    assert.equal(isRoleCovered('@org/aws-reviewers', commenterRoles, codeownerMappings), false);
  });

  test('empty commenter roles returns false', () => {
    assert.equal(isRoleCovered('@org/aws-reviewers', new Set(), codeownerMappings), false);
  });
});

// ---------------------------------------------------------------------------
// findUncoveredRoles
// ---------------------------------------------------------------------------

describe('findUncoveredRoles', () => {
  const codeownerMappings = [
    { pattern: 'pkg/', roles: ['@org/pkg-reviewers'] },
    { pattern: 'pkg/provider/aws/', roles: ['@org/aws-reviewers'] },
    { pattern: 'pkg/provider/gcp/', roles: ['@org/gcp-reviewers'] },
  ];

  test('returns empty array when all roles are covered', () => {
    const required = new Set(['@org/aws-reviewers', '@org/gcp-reviewers']);
    const commenter = new Set(['@org/pkg-reviewers']); // covers both
    const uncovered = findUncoveredRoles(required, commenter, codeownerMappings);
    assert.deepEqual(uncovered, []);
  });

  test('returns uncovered role when commenter lacks coverage', () => {
    const codeownerMappingsLocal = [
      { pattern: 'pkg/provider/aws/', roles: ['@org/aws-reviewers'] },
      { pattern: 'pkg/provider/gcp/', roles: ['@org/gcp-reviewers'] },
    ];
    const required = new Set(['@org/aws-reviewers', '@org/gcp-reviewers']);
    const commenter = new Set(['@org/aws-reviewers']);
    const uncovered = findUncoveredRoles(required, commenter, codeownerMappingsLocal);
    assert.deepEqual(uncovered, ['@org/gcp-reviewers']);
  });

  test('returns all roles when commenter has no matching roles', () => {
    const required = new Set(['@org/aws-reviewers']);
    const uncovered = findUncoveredRoles(required, new Set(), codeownerMappings);
    assert.deepEqual(uncovered, ['@org/aws-reviewers']);
  });
});

// ---------------------------------------------------------------------------
// buildConfirmationMessage
// ---------------------------------------------------------------------------

describe('buildConfirmationMessage', () => {
  const org = 'external-secrets';
  const wildcardRoles = ['@external-secrets/maintainers', '@external-secrets/approvers'];

  test('all roles covered produces success message', () => {
    const commenterRoles = new Set(['@external-secrets/aws-reviewers']);
    const requiredRoles = new Set(['@external-secrets/aws-reviewers']);
    const msg = buildConfirmationMessage(
      'alice',
      org,
      commenterRoles,
      requiredRoles,
      [],
      wildcardRoles
    );
    assert.ok(msg.includes('✅ LGTM by @alice'));
    assert.ok(msg.includes('✅ All required roles covered'));
    assert.ok(!msg.includes('❌'));
  });

  test('uncovered roles listed in message', () => {
    const commenterRoles = new Set(['@external-secrets/aws-reviewers']);
    const requiredRoles = new Set([
      '@external-secrets/aws-reviewers',
      '@external-secrets/gcp-reviewers',
    ]);
    const uncovered = ['@external-secrets/gcp-reviewers'];
    const msg = buildConfirmationMessage(
      'bob',
      org,
      commenterRoles,
      requiredRoles,
      uncovered,
      wildcardRoles
    );
    assert.ok(msg.includes('✅ LGTM by @bob'));
    assert.ok(msg.includes('❌ Additional review may be needed by:'));
    assert.ok(msg.includes('gcp-reviewers'));
    assert.ok(msg.includes('aws-reviewers'));
  });

  test('wildcard roles are mentioned at the top of the message', () => {
    const msg = buildConfirmationMessage(
      'carol',
      org,
      new Set(['@external-secrets/maintainers']),
      new Set(['@external-secrets/maintainers']),
      [],
      wildcardRoles
    );
    assert.ok(msg.startsWith('@external-secrets/maintainers @external-secrets/approvers'));
  });

  test('role names are stripped of org prefix in message body', () => {
    const commenterRoles = new Set(['@external-secrets/aws-reviewers']);
    const requiredRoles = new Set([
      '@external-secrets/aws-reviewers',
      '@external-secrets/gcp-reviewers',
    ]);
    const uncovered = ['@external-secrets/gcp-reviewers'];
    const msg = buildConfirmationMessage(
      'dave',
      org,
      commenterRoles,
      requiredRoles,
      uncovered,
      wildcardRoles
    );
    // The body should use short names, not full @org/name strings
    assert.ok(msg.includes('- aws-reviewers'));
    assert.ok(msg.includes('- gcp-reviewers'));
  });
});
