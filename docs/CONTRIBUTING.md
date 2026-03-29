# Contributing to mroki

Thanks for your interest in contributing to mroki! This guide will help you get started.

## Getting Started

1. **Fork** the repository on GitHub
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/mroki.git
   cd mroki
   ```
3. **Set up** your development environment — see the [Development Guide](guides/DEVELOPMENT.md)

## Development Workflow

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```
2. Make your changes
3. Write or update tests for your changes
4. Run the test suite:
   ```bash
   # Go
   go test -race ./...

   # Hub (Vue)
   cd web/mroki-hub && pnpm test
   ```
5. Commit using [Conventional Commits](#commit-messages)
6. Push to your fork and open a Pull Request

## Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**

| Type | Description |
|---|---|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `test` | Adding or updating tests |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `chore` | Maintenance (deps, CI, build) |

**Scopes:** `proxy`, `api`, `hub`, `caddy`, `diff`, `docs`

**Examples:**
```
feat(proxy): add retry logic with exponential backoff
fix(api): handle nil pointer in gate creation
docs: update API contracts documentation
test(proxy): add test for timeout handling
refactor(hub): restrict diff split view to md+ screens
```

## Pull Request Guidelines

- Keep PRs focused — one feature or fix per PR
- Include tests for new functionality
- Update documentation if behavior changes
- Make sure all tests pass before requesting review
- Link related issues in the PR description

## Code Style

### Go
- Follow standard `gofmt` formatting
- Use `golangci-lint` for linting: `make lint-api` / `make lint-proxy`
- Write table-driven tests where appropriate
- Keep functions short and focused

### Vue / TypeScript (mroki-hub)
- TypeScript is required in all Vue components (`<script setup lang="ts">`)
- Use Composition API with `<script setup>`
- Follow the existing Tailwind CSS patterns
- Run `pnpm lint` and `pnpm format` before committing

## Reporting Bugs

Open a [GitHub Issue](https://github.com/pedrobarco/mroki/issues) with:

- A clear title and description
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, browser)

## Suggesting Features

Open a [GitHub Issue](https://github.com/pedrobarco/mroki/issues) with:

- A clear use case — what problem does it solve?
- Proposed solution (if any)
- Alternatives you've considered

## Project Structure

See the [Development Guide](guides/DEVELOPMENT.md#project-structure) for a full breakdown of the repository layout.

## Questions?

- Check the [documentation](./README.md)
- Open a [discussion](https://github.com/pedrobarco/mroki/discussions) on GitHub
