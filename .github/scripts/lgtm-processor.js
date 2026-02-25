'use strict';

const fs = require('fs');

const ORGANIZATION = 'external-secrets';
const LGTM_LABEL = 'lgtm';

/**
 * Parse CODEOWNERS.md content into role mappings.
 * @param {string} content - The CODEOWNERS.md file content
 * @param {string} organization - The GitHub organization name
 * @returns {{ codeownerMappings: Array<{pattern: string, roles: string[]}>, wildcardRoles: string[], maintainerRoles: string[] }}
 */
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
        rolesList.forEach(r => { if (!wildcardRoles.includes(r)) wildcardRoles.push(r); });
      } else {
        codeownerMappings.push({ pattern, roles: rolesList });
      }
    }
  });

  const maintainerRoles = wildcardRoles.map(role => role.replace(`@${organization}/`, ''));
  return { codeownerMappings, wildcardRoles, maintainerRoles };
}

/**
 * Check whether a file path matches a codeowner pattern.
 * Supports exact matches and hierarchical directory prefix matches.
 * @param {string} file - The file path to check
 * @param {string} pattern - The codeowner pattern
 * @returns {boolean}
 */
function fileMatchesPattern(file, pattern) {
  return (
    file === pattern ||
    file.startsWith(pattern.endsWith('/') ? pattern : pattern + '/')
  );
}

/**
 * Determine the set of required reviewer roles for a list of changed files.
 * Files with no specific owner fall back to wildcard roles.
 * @param {string[]} changedFiles - List of changed file paths
 * @param {Array<{pattern: string, roles: string[]}>} codeownerMappings
 * @param {string[]} wildcardRoles
 * @returns {{ requiredReviewerRoles: Set<string>, hasFilesWithoutSpecificOwners: boolean }}
 */
function getRequiredReviewerRoles(changedFiles, codeownerMappings, wildcardRoles) {
  const requiredReviewerRoles = new Set();
  let hasFilesWithoutSpecificOwners = false;

  changedFiles.forEach(file => {
    let hasSpecificOwner = false;
    codeownerMappings.forEach(mapping => {
      if (fileMatchesPattern(file, mapping.pattern)) {
        mapping.roles.forEach(role => { requiredReviewerRoles.add(role); });
        hasSpecificOwner = true;
      }
    });
    if (!hasSpecificOwner) {
      hasFilesWithoutSpecificOwners = true;
    }
  });

  if (hasFilesWithoutSpecificOwners) {
    wildcardRoles.forEach(role => { requiredReviewerRoles.add(role); });
  }

  return { requiredReviewerRoles, hasFilesWithoutSpecificOwners };
}

/**
 * Check whether a required reviewer role is covered by the commenter's roles,
 * including hierarchical coverage (a broader pattern role covers a more specific one).
 * @param {string} requiredRole
 * @param {Set<string>} commenterReviewerRoles
 * @param {Array<{pattern: string, roles: string[]}>} codeownerMappings
 * @returns {boolean}
 */
function isRoleCovered(requiredRole, commenterReviewerRoles, codeownerMappings) {
  if (commenterReviewerRoles.has(requiredRole)) return true;

  const requiredRolePatterns = codeownerMappings
    .filter(mapping => mapping.roles.includes(requiredRole))
    .map(mapping => mapping.pattern);

  for (const commenterRole of commenterReviewerRoles) {
    const commenterRolePatterns = codeownerMappings
      .filter(mapping => mapping.roles.includes(commenterRole))
      .map(mapping => mapping.pattern);

    for (const commenterPattern of commenterRolePatterns) {
      for (const requiredPattern of requiredRolePatterns) {
        const commenterPath = commenterPattern.endsWith('/') ? commenterPattern : commenterPattern + '/';
        const requiredPath = requiredPattern.endsWith('/') ? requiredPattern : requiredPattern + '/';

        // If the required pattern is a subdirectory of the commenter pattern, it is covered
        if (requiredPath.startsWith(commenterPath) && commenterPath !== requiredPath) {
          return true;
        }
      }
    }
  }

  return false;
}

/**
 * Find all required reviewer roles not covered by the commenter's roles.
 * @param {Set<string>} requiredReviewerRoles
 * @param {Set<string>} commenterReviewerRoles
 * @param {Array<{pattern: string, roles: string[]}>} codeownerMappings
 * @returns {string[]}
 */
function findUncoveredRoles(requiredReviewerRoles, commenterReviewerRoles, codeownerMappings) {
  const uncoveredRoles = [];
  for (const requiredRole of requiredReviewerRoles) {
    if (!isRoleCovered(requiredRole, commenterReviewerRoles, codeownerMappings)) {
      uncoveredRoles.push(requiredRole);
    }
  }
  return uncoveredRoles;
}

/**
 * Build the confirmation comment body for a successful LGTM.
 * @param {string} commenter
 * @param {string} organization
 * @param {Set<string>} commenterReviewerRoles
 * @param {Set<string>} requiredReviewerRoles
 * @param {string[]} uncoveredRoles
 * @param {string[]} wildcardRoles
 * @returns {string}
 */
function buildConfirmationMessage(
  commenter,
  organization,
  commenterReviewerRoles,
  requiredReviewerRoles,
  uncoveredRoles,
  wildcardRoles
) {
  const mentionRoles = wildcardRoles.join(' ');
  let message = `${mentionRoles}\n\n✅ LGTM by @${commenter}`;

  message += `\n\n**Review Coverage:**`;

  if (uncoveredRoles.length > 0) {
    message += `\n- Commenter has roles:`;
    Array.from(commenterReviewerRoles).forEach(role => {
      message += `\n  - ${role.replace(`@${organization}/`, '')}`;
    });
    message += `\n- Required roles:`;
    Array.from(requiredReviewerRoles).forEach(role => {
      message += `\n  - ${role.replace(`@${organization}/`, '')}`;
    });
    message += `\n- ❌ Additional review may be needed by:`;
    uncoveredRoles.forEach(role => {
      message += `\n  - ${role.replace(`@${organization}/`, '')}`;
    });
  } else {
    message += `\n- ✅ All required roles covered`;
  }

  return message;
}

/**
 * Main LGTM processor – handles the /lgtm command on pull requests.
 * Intended to be called from actions/github-script via dynamic import.
 * @param {{ core: object, github: object, context: object }} params
 */
async function lgtmProcessor({ core, github, context }) {
  const commenter = context.payload.comment.user.login;
  const prNumber = context.payload.issue.number;
  const owner = context.repo.owner;
  const repo = context.repo.repo;

  // Verify the LGTM label exists in the repository before proceeding
  try {
    await github.rest.issues.getLabel({
      owner, repo, name: LGTM_LABEL,
    });
  } catch (error) {
    if (error.status === 404) {
      core.setFailed(
        `Label "${LGTM_LABEL}" does not exist in ${owner}/${repo}. ` +
        `Please create the label before using /lgtm.`
      );
      return;
    }
    core.warning(`Failed to check label existence: ${error.message}`);
  }

  // Parse CODEOWNERS.md file
  let codeownersContent;
  try {
    codeownersContent = fs.readFileSync('CODEOWNERS.md', 'utf8');
  } catch (error) {
    core.warning('Could not read CODEOWNERS.md: ' + error.message);
    return;
  }

  const { codeownerMappings, wildcardRoles, maintainerRoles } = parseCodeowners(
    codeownersContent,
    ORGANIZATION
  );

  // Early check: if the commenter is a maintainer, approve immediately
  let isMaintainer = false;
  for (const role of maintainerRoles) {
    try {
      const response = await github.rest.teams.getMembershipForUserInOrg({
        org: ORGANIZATION,
        team_slug: role,
        username: commenter,
      });
      if (response.data.state === 'active') {
        isMaintainer = true;
        break;
      }
    } catch (error) {
      if (error.status !== 404) {
        core.warning(`Failed to check team membership for ${commenter} in ${role}: ${error.message}`);
      }
    }
  }

  if (isMaintainer) {
    const labels = await github.rest.issues.listLabelsOnIssue({
      owner, repo, issue_number: prNumber,
    });
    if (!labels.data.some(l => l.name === LGTM_LABEL)) {
      await github.rest.issues.addLabels({
        owner, repo, issue_number: prNumber, labels: [LGTM_LABEL],
      });
    }
    await github.rest.issues.createComment({
      owner, repo, issue_number: prNumber,
      body: `✅ LGTM by @${commenter} (maintainer)`,
    });
    return;
  }

  // Get the list of files changed by the PR (paginated to handle 30+ files)
  const allFiles = await github.paginate(github.rest.pulls.listFiles, {
    owner, repo, pull_number: prNumber, per_page: 100,
  });
  const changedFiles = allFiles.map(f => f.filename);

  const { requiredReviewerRoles } = getRequiredReviewerRoles(
    changedFiles,
    codeownerMappings,
    wildcardRoles
  );

  // Determine which of the required roles the commenter belongs to
  const commenterReviewerRoles = new Set();
  for (const role of requiredReviewerRoles) {
    const roleSlug = role.replace(`@${ORGANIZATION}/`, '');
    try {
      const response = await github.rest.teams.getMembershipForUserInOrg({
        org: ORGANIZATION,
        team_slug: roleSlug,
        username: commenter,
      });
      if (response.data.state === 'active') {
        commenterReviewerRoles.add(role);
      }
    } catch (error) {
      if (error.status !== 404) {
        core.warning(`Failed to check role membership for ${commenter} in ${roleSlug}: ${error.message}`);
      }
    }
  }

  // Reject if commenter has none of the required roles (skip if no roles required)
  if (requiredReviewerRoles.size === 0) {
    core.info('No required reviewer roles for this PR, skipping role check');
  } else if (commenterReviewerRoles.size === 0) {
    const rolesList = Array.from(requiredReviewerRoles)
      .map(role => role.replace(`@${ORGANIZATION}/`, ''))
      .join(', ');
    await github.rest.issues.createComment({
      owner, repo, issue_number: prNumber,
      body:
        `@${commenter} You must be a member of one of the required reviewer roles to use /lgtm.\n\n` +
        `Required roles for this PR: ${rolesList}`,
    });
    return;
  }

  // Add LGTM label if not already present
  const labels = await github.rest.issues.listLabelsOnIssue({
    owner, repo, issue_number: prNumber,
  });
  if (!labels.data.some(l => l.name === LGTM_LABEL)) {
    await github.rest.issues.addLabels({
      owner, repo, issue_number: prNumber, labels: [LGTM_LABEL],
    });
  }

  // Build and post the confirmation comment
  const uncoveredRoles = findUncoveredRoles(
    requiredReviewerRoles,
    commenterReviewerRoles,
    codeownerMappings
  );
  const confirmationMessage = buildConfirmationMessage(
    commenter,
    ORGANIZATION,
    commenterReviewerRoles,
    requiredReviewerRoles,
    uncoveredRoles,
    wildcardRoles
  );

  await github.rest.issues.createComment({
    owner, repo, issue_number: prNumber,
    body: confirmationMessage,
  });
}

module.exports = lgtmProcessor;
module.exports.parseCodeowners = parseCodeowners;
module.exports.fileMatchesPattern = fileMatchesPattern;
module.exports.getRequiredReviewerRoles = getRequiredReviewerRoles;
module.exports.isRoleCovered = isRoleCovered;
module.exports.findUncoveredRoles = findUncoveredRoles;
module.exports.buildConfirmationMessage = buildConfirmationMessage;
