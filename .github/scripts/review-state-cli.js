#!/usr/bin/env node
/**
 * Local dry run for the review state router.
 *
 * Reads the live repository and prints the label each open pull request would
 * get. It cannot write: the client's mutating methods throw, so a plumbing bug
 * surfaces as a crash here instead of a change to a real pull request.
 *
 * Usage:
 *   node .github/scripts/review-state-cli.js
 *   node .github/scripts/review-state-cli.js --pr 6613 --pr 6851
 *   node .github/scripts/review-state-cli.js --repo owner/name --json
 *
 * Auth: GH_TOKEN or GITHUB_TOKEN, otherwise `gh auth token` is used.
 */

import { execFileSync } from 'node:child_process';
import run from './review-state.js';

const API = 'https://api.github.com';

function parseArgs(argv) {
  const args = { repo: 'external-secrets/external-secrets', prs: [], json: false };
  for (let i = 0; i < argv.length; i += 1) {
    const a = argv[i];
    if (a === '--pr') {
      const raw = argv[++i];
      const number = Number(raw);
      if (!Number.isSafeInteger(number) || number <= 0) {
        throw new Error(`--pr needs a positive pull request number, got ${JSON.stringify(raw)}`);
      }
      args.prs.push(number);
    } else if (a === '--repo') {
      args.repo = argv[++i];
    } else if (a === '--json') {
      args.json = true;
    } else if (a === '--help' || a === '-h') {
      args.help = true;
    } else {
      throw new Error(`Unknown argument: ${a}`);
    }
  }
  return args;
}

function token() {
  if (process.env.GH_TOKEN) return process.env.GH_TOKEN;
  if (process.env.GITHUB_TOKEN) return process.env.GITHUB_TOKEN;
  try {
    return execFileSync('gh', ['auth', 'token'], { encoding: 'utf8' }).trim();
  } catch {
    throw new Error('No token. Set GH_TOKEN, or run `gh auth login`.');
  }
}

function makeClient(auth) {
  const headers = {
    authorization: `Bearer ${auth}`,
    accept: 'application/vnd.github+json',
    'user-agent': 'eso-review-state-cli',
  };

  async function request(path, { method = 'GET', body } = {}) {
    const res = await fetch(path.startsWith('http') ? path : `${API}${path}`, {
      method,
      headers: body ? { ...headers, 'content-type': 'application/json' } : headers,
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) {
      const error = new Error(`${method} ${path} -> ${res.status} ${await res.text()}`);
      error.status = res.status;
      throw error;
    }
    return res.json();
  }

  // Anything that would mutate the repository is deliberately unavailable, so
  // "dry run" is enforced by the client rather than by remembering a flag.
  const readOnly = (name) => () => {
    throw new Error(`${name} is not available in the local dry run`);
  };

  const endpoint = (template) => {
    const fn = async (params) => {
      const query = new URLSearchParams();
      const path = template.replace(/\{(\w+)\}/g, (_, k) => encodeURIComponent(params[k]));
      for (const [k, v] of Object.entries(params)) {
        if (!template.includes(`{${k}}`) && k !== 'owner' && k !== 'repo') query.set(k, v);
      }
      const qs = query.toString();
      return { data: await request(qs ? `${path}?${qs}` : path) };
    };
    fn.__template = template;
    return fn;
  };

  const github = {
    graphql: async (query, variables) => {
      const out = await request('/graphql', { method: 'POST', body: { query, variables } });
      if (out.errors) throw new Error(`GraphQL: ${JSON.stringify(out.errors)}`);
      return out.data;
    },
    paginate: async (fn, params) => {
      const all = [];
      for (let page = 1; ; page += 1) {
        const { data } = await fn({ ...params, page });
        if (!Array.isArray(data) || data.length === 0) break;
        all.push(...data);
        if (data.length < (params.per_page || 30)) break;
      }
      return all;
    },
    rest: {
      issues: {
        listComments: endpoint('/repos/{owner}/{repo}/issues/{issue_number}/comments'),
        listLabelsForRepo: endpoint('/repos/{owner}/{repo}/labels'),
        listLabelsOnIssue: endpoint('/repos/{owner}/{repo}/issues/{issue_number}/labels'),
        addLabels: readOnly('addLabels'),
        removeLabel: readOnly('removeLabel'),
        createComment: readOnly('createComment'),
        updateComment: readOnly('updateComment'),
      },
      pulls: { list: endpoint('/repos/{owner}/{repo}/pulls') },
    },
  };
  return github;
}

const core = {
  info: (m) => process.stderr.write(`  ${m}\n`),
  warning: (m) => process.stderr.write(`  WARN ${m}\n`),
  error: (m) => process.stderr.write(`  ERROR ${m}\n`),
  setFailed: (m) => process.stderr.write(`  FAILED ${m}\n`),
};

function table(results) {
  const rows = results.map((r) => ({
    pr: `#${r.number}`,
    state: r.state.replace('review/', ''),
    now: (r.current.filter((l) => l.startsWith('review/'))[0] || '-').replace('review/', ''),
    appr: `${r.approvals}/${r.required}`,
    bots: String(r.botFindingsOpen),
    comment: r.commentAction,
  }));
  const cols = ['pr', 'state', 'now', 'appr', 'bots', 'comment'];
  const width = {};
  for (const c of cols) width[c] = Math.max(c.length, ...rows.map((r) => r[c].length));
  const line = (r) => cols.map((c) => String(r[c]).padEnd(width[c])).join('  ');
  const out = [line(Object.fromEntries(cols.map((c) => [c, c.toUpperCase()])))];
  out.push(cols.map((c) => '-'.repeat(width[c])).join('  '));
  for (const r of rows) out.push(line(r));
  return out.join('\n');
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  if (args.help) {
    process.stdout.write(`${new URL(import.meta.url).pathname}\n`
      + '  --pr N     evaluate one pull request, repeatable\n'
      + '  --repo O/R target repository\n'
      + '  --json     machine readable output\n');
    return;
  }

  const parts = (args.repo || '').split('/');
  if (parts.length !== 2 || !parts[0] || !parts[1]) {
    throw new Error(`--repo must be exactly owner/name, got ${JSON.stringify(args.repo)}`);
  }
  const [owner, repo] = parts;

  const github = makeClient(token());
  const context = { repo: { owner, repo }, eventName: 'workflow_dispatch', payload: {} };

  let numbers = args.prs;
  if (numbers.length === 0) {
    const open = await github.paginate(github.rest.pulls.list, {
      owner, repo, state: 'open', per_page: 100,
    });
    numbers = open.map((p) => p.number);
    core.info(`sweeping ${numbers.length} open pull requests`);
  }

  const results = await run({ core, github, context, numbers, dryRun: true });

  if (args.json) {
    process.stdout.write(`${JSON.stringify(results, null, 2)}\n`);
    return;
  }
  process.stdout.write(`\n${table(results)}\n\n`);
  const counts = results.reduce((acc, r) => {
    acc[r.state] = (acc[r.state] || 0) + 1;
    return acc;
  }, {});
  for (const [state, n] of Object.entries(counts).sort((a, b) => b[1] - a[1])) {
    process.stdout.write(`  ${String(n).padStart(3)}  ${state}\n`);
  }
  const changes = results.filter((r) => r.labelChange.added || r.labelChange.removed.length);
  process.stdout.write(`\n  ${changes.length} of ${results.length} would change label\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.message}\n`);
  process.exit(1);
});
