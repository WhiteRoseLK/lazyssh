# Contributing to neossh

Thank you for your interest in contributing to **neossh**! We welcome bug reports, feature requests, documentation improvements, and code contributions.

---

## 🛠️ Development Setup

### Prerequisites
- [Go](https://go.dev/dl/) 1.24+
- `make`

### Useful Make Commands

| Command | Description |
|---|---|
| `make build` | Formats code, lints, and builds the `neossh` binary in `./bin/neossh` |
| `make test` | Runs unit tests with race detection and generates coverage profile |
| `make test-verbose` | Runs unit tests with verbose test logs |
| `make lint` | Runs `golangci-lint` to check code quality |
| `make fmt` | Formats code with `gofumpt` and standard Go formatting |
| `make clean` | Cleans build caches and output directory |

---

## 🌿 Branching & Pull Requests

1. **Trunk-Based Workflow**:
   - `main` is our production branch.
   - All contributions should be developed in dedicated feature or bugfix branches (e.g., `feat/clipboard-paste` or `fix/reboot-hang`).

2. **Branch Naming**:
   - `feat/<short-name>`
   - `fix/<short-name>`
   - `docs/<short-name>`
   - `refactor/<short-name>`

3. **Submitting a Pull Request**:
   - Open a PR targeting `main`.
   - Fill in the PR template, linking the relevant issue (e.g. `Closes #6`).
   - Ensure the CI checks (`lint`, `test`, `build`) pass.

---

## 💬 Conventional Commits

We use **Conventional Commits** to automate our versioning, changelog generation, and releases via **Google Release Please**.

Commit messages and PR titles should follow this format:

```text
<type>(<scope>): <short description>
```

### Common Types:
- `feat`: A new feature (triggers a **minor** release `v1.X.0`)
- `fix`: A bug fix (triggers a **patch** release `v1.0.X`)
- `docs`: Documentation changes
- `chore`: Maintenance tasks, dependency updates
- `refactor`: Code changes that neither fix a bug nor add a feature
- `test`: Adding or updating tests
- `ci`: CI/CD workflow updates

### Breaking Changes:
Add `!` after type/scope or include `BREAKING CHANGE:` in the commit body (triggers a **major** release `vX.0.0`).

---

## 🚀 Release Process

Releases are fully automated with **Release Please** and **GoReleaser**:
1. When pull requests are merged into `main`, Release Please maintains an open **Release PR** that collects changes into `CHANGELOG.md` and bumps the version.
2. When the maintainer merges the Release PR, Release Please automatically creates the Git tag and GitHub Release.
3. This triggers **GoReleaser**, which compiles multi-platform binaries and updates the **Homebrew Tap** automatically.
