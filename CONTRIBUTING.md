# Contributing to Crossbeam

Thank you for your interest in contributing to Crossbeam! This document outlines everything you need to know to get your contribution merged smoothly.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Ways to Contribute](#ways-to-contribute)
- [Reporting Bugs](#reporting-bugs)
- [Requesting Features](#requesting-features)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Branching Strategy](#branching-strategy)
- [Commit Messages](#commit-messages)
- [Code Style](#code-style)
- [Dependency Licensing Policy](#dependency-licensing-policy)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Review Criteria](#review-criteria)
- [Security Vulnerabilities](#security-vulnerabilities)
- [Getting Help](#getting-help)

---

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md). We are committed to making participation in this project a welcoming, respectful experience for everyone.

---

## Ways to Contribute

You don't need to write code to contribute. Here are several ways you can help:

- **Report bugs** — Help us identify unexpected behavior
- **Suggest features** — Propose improvements or new functionality
- **Improve documentation** — Fix typos, clarify explanations, add examples
- **Write tests** — Increase coverage for existing features
- **Review pull requests** — Provide constructive feedback on open PRs
- **Triage issues** — Help label, categorize, and reproduce reported issues
- **Spread the word** — Star the repo, share it with others

---

## Reporting Bugs

Before opening a bug report, please:

1. **Search existing issues** to avoid duplicates
2. **Check if it's already fixed** on the `main` branch
3. **Reproduce it with the latest version** of the project

When filing a bug report, include:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected behavior vs. actual behavior
- Relevant logs, screenshots, or screen recordings
- Your environment (OS, client version, Bun version, etc.)

Use the [**Bug Report** issue template](.github/ISSUE_TEMPLATE/bug_report.md) when creating the issue — it will be offered automatically when you open a new issue on GitHub.

---

## Requesting Features

Feature requests are welcome. To help us understand and prioritize your idea:

1. **Search existing issues** — your idea may already be requested or in progress
2. **Describe the problem** you're trying to solve, not just the solution
3. **Explain the use case** — who benefits and how often?
4. **Consider the scope** — is this a small enhancement or a significant new subsystem?

Use the [**Feature Request** issue template](.github/ISSUE_TEMPLATE/feature_request.md) when creating the issue — it will be offered automatically when you open a new issue on GitHub. Large proposals may be asked to go through a lightweight RFC (Request for Comments) process before implementation begins.

---

## Development Setup

### Prerequisites

| Tool | Version |
|------|---------|
| Bun | latest |
| Docker | `28.x+` |
| Docker Compose | `5.x+` |

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:

   ```bash
   git clone https://github.com/<your-username>/crossbeam.git
   cd crossbeam
   ```

3. Add the upstream remote:

   ```bash
   git remote add upstream https://github.com/low-stack-technologies/crossbeam.git
   ```

### Install Dependencies

```bash
bun install
```

### Environment Setup

```bash
cp .env.example .env
# Edit .env with your local configuration
```

### Start Infrastructure

```bash
docker compose up postgres redis minio -d
```

### Run Migrations

```bash
bun db:migrate
```

### Start the Development Server

```bash
bun dev
```

The application will be available at [http://localhost:3000](http://localhost:3000).

---

## Project Structure

```
crossbeam/
├── apps/
│   ├── desktop/    # Electron desktop client (React)
│   └── server/     # Bun backend + WebSocket gateway
├── packages/
│   ├── shared/     # Types and utilities shared across apps
│   └── config/     # Shared tooling configuration
├── docker/         # Docker and Compose configuration
└── .github/
    ├── ISSUE_TEMPLATE/   # Bug report and feature request templates
    └── workflows/        # CI/CD pipelines
```

Each `apps/*` and `packages/*` directory is an independent workspace with its own `package.json`. Most commands can be run from the monorepo root using `bun --filter`.

---

## Branching Strategy

We use a simple trunk-based branching model:

| Branch | Purpose |
|--------|---------|
| `main` | Stable, releasable code. Protected. |
| `feature/<name>` | New features |
| `fix/<name>` | Bug fixes |
| `docs/<name>` | Documentation-only changes |
| `chore/<name>` | Tooling, CI, dependency updates |
| `refactor/<name>` | Code refactoring with no behavior change |

**Always branch from `main`:**

```bash
git fetch upstream
git checkout -b feature/my-feature upstream/main
```

Never commit directly to `main`.

---

## Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification. A well-formed commit message looks like:

```
<type>(<scope>): <short summary>

<optional body — explain what and why, not how>

<optional footer — breaking changes, issue refs>
```

### Types

| Type | When to use |
|------|-------------|
| `feat` | A new feature |
| `fix` | A bug fix |
| `docs` | Documentation only changes |
| `style` | Formatting — no logic change |
| `refactor` | Code refactor — no feature or fix |
| `perf` | Performance improvement |
| `test` | Adding or updating tests |
| `chore` | Build process, dependencies, tooling |
| `ci` | CI/CD configuration |

### Examples

```
feat(push): add targeted push to a single device

fix(ws): handle reconnect on unexpected gateway close

docs(readme): add self-hosting deployment instructions

chore(deps): upgrade electron to 32.0.0
```

Breaking changes must be noted in the footer:

```
feat(api)!: rename /pushes to /api/v1/pushes

BREAKING CHANGE: existing clients must update all push endpoint paths
```

---

## Code Style

The project uses automated formatters and linters. Your code must pass all checks before it can be merged.

| Tool | Purpose |
|------|---------|
| **ESLint** | TypeScript linting |
| **Prettier** | Code formatting |
| **TypeScript** | Static type checking |

### Lint & Format

```bash
# Lint all packages
bun lint

# Auto-fix lint issues
bun lint:fix

# Format all files
bun format

# Type-check all packages
bun typecheck
```

### Guidelines

- Prefer `const` over `let`; never use `var`
- Prefer named exports over default exports
- Keep functions small and focused on a single responsibility
- Avoid `any` — use proper types or `unknown` with narrowing
- Do not suppress TypeScript errors with `@ts-ignore` unless you document why
- Co-locate tests next to the source files they test (`foo.ts` → `foo.test.ts`)

---

## Dependency Licensing Policy

Crossbeam is MIT licensed and we take license compatibility seriously. All dependencies — runtime and development — must carry a license that is compatible with MIT distribution.

### Permitted licenses

The following licenses are acceptable without any further approval:

| License | Notes |
|---------|-------|
| MIT | Preferred |
| ISC | Functionally equivalent to MIT |
| BSD-2-Clause | Permitted |
| BSD-3-Clause | Permitted |
| Apache-2.0 | Permitted — note: incompatible with GPLv2, but fine for MIT projects |
| CC0-1.0 | Public domain dedication — permitted |
| Unlicense | Public domain dedication — permitted |
| 0BSD | Permitted |

### Requires maintainer approval

These licenses impose additional conditions that need case-by-case review before a dependency can be merged:

| License | Reason for review |
|---------|------------------|
| MPL-2.0 | File-level copyleft; generally fine but must be evaluated per use |
| LGPL-2.1 / LGPL-3.0 | Copyleft with linking exception; dynamic linking only — static bundling is not permitted |
| CC-BY-4.0 | Acceptable for assets/fonts, not for code |

### Not permitted

The following licenses are **incompatible** with Crossbeam's MIT license and may not be introduced under any circumstances:

- GPL-2.0, GPL-3.0 (and any GPL variant without a linking exception)
- AGPL-3.0
- SSPL
- Commons Clause additions
- Any proprietary or source-available license

### How to check

Before adding a new package, verify its license:

```bash
# Check the license field of a specific package
bun pm ls --json | grep -A2 '"your-package"'
```

For a full audit of the dependency tree you can use a tool such as [license-checker](https://github.com/davglass/license-checker) (MIT licensed).

If you are unsure whether a license is compatible, open a GitHub Discussion before adding the dependency. Do not merge first and ask later.

---

## Testing

We aim for a high degree of test coverage across all critical paths.

### Running Tests

```bash
# Run all tests
bun test

# Run tests in watch mode
bun test --watch

# Run tests with coverage
bun test --coverage

# Run end-to-end tests
bun test:e2e
```

### Writing Tests

- **Unit tests** — use Bun's built-in test runner for pure functions and isolated modules
- **Integration tests** — use Bun test with a real database (no mocking the DB layer)
- **End-to-end tests** — use Playwright for full UI flows in the Electron client

Every non-trivial PR should include tests for the changed behavior. Bug fixes should include a regression test that fails before the fix and passes after.

---

## Pull Request Process

### Before Opening a PR

- [ ] Your branch is up to date with `upstream/main`
- [ ] `bun lint` passes
- [ ] `bun typecheck` passes
- [ ] `bun test` passes
- [ ] New or changed behavior is covered by tests
- [ ] Documentation is updated if the public API or behavior changed

### Keeping Your Branch Up to Date

```bash
git fetch upstream
git rebase upstream/main
```

Prefer rebasing over merging to keep history clean.

### Opening the PR

- Fill out the **Pull Request Template** completely
- Link the related issue(s) using `Closes #123` or `Fixes #123` in the PR description
- Keep PRs focused — one logical change per PR
- Mark the PR as a **Draft** if it's not yet ready for review
- Add screenshots or recordings for any UI changes

### After Opening the PR

- CI will run automatically. Fix any failures before requesting a review.
- A maintainer will be assigned to review your PR.
- Address review feedback with new commits (do not force-push during active review).
- Once approved, a maintainer will squash-merge your PR into `main`.

---

## Review Criteria

Reviewers will evaluate your contribution against the following:

| Criterion | What we look for |
|-----------|-----------------|
| **Correctness** | Does the change do what it claims? Are edge cases handled? |
| **Tests** | Is behavior covered by automated tests? |
| **Performance** | Does the change introduce unnecessary work on hot paths? |
| **Security** | No new injection vectors, insecure defaults, or leaked secrets |
| **Backwards compatibility** | Breaking changes must be documented and intentional |
| **Code clarity** | Is the code easy to read and reason about? |
| **Scope** | Does the PR stay focused on the stated goal? |

---

## Security Vulnerabilities

**Please do not open public GitHub issues for security vulnerabilities.**

If you discover a security issue, disclose it responsibly by emailing **security@low-stack.tech**. Include:

- A description of the vulnerability
- Steps to reproduce or a proof-of-concept
- The potential impact
- Any suggested mitigations

We will acknowledge receipt within 48 hours and aim to ship a fix within 14 days of disclosure. You will be credited in the release notes unless you request otherwise.

---

## Getting Help

If you're stuck or have questions:

- **GitHub Discussions** — For open-ended questions, design discussions, and help getting started
- **GitHub Issues** — For confirmed bugs and actionable feature requests

We appreciate your effort and look forward to reviewing your contribution!
