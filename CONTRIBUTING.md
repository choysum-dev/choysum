# Contributing to Choysum

Thank you for your interest in contributing! To maintain the project's health and legal integrity, please follow these guidelines.

---

## ⚖️ Licensing Model

Choysum uses a dual-license architecture to balance core protection with ecosystem flexibility.

| Component | License | Directory Scope |
| :--- | :--- | :--- |
| **Core Framework** | LGPL-3.0-or-later | Everything outside `modules/` |
| **Official Modules**| Apache-2.0 | All files within `modules/` |

**Important:** The project owner reserves the right to sublicense contributions under commercial terms (Dual-licensing). By contributing, you agree to this strategy.

---

## 📝 Coding Standards

### SPDX Headers
Every new source files should include an SPDX license identifier at the top:

- **Core files:** `// SPDX-License-Identifier: LGPL-3.0-or-later`
- **Module files:** `// SPDX-License-Identifier: Apache-2.0`

### Boundary Rules
- **No Leakage:** Do not copy or "vendor" core framework code into the `modules/` directory.
- **Dependency Check:** Check third-party license compatibility before adding dependencies.
- **Go Style:** Please run `go fmt ./...` before submitting.

### CLI Test Semantics Contract
The `test unit`, `test typecheck`, and `test e2e` commands share one semantics source of truth for unknown-target, no-tests-found, and command-prefix behavior.

| Command | Runtime Options Prefix | Unknown Explicit Target | Existing Target With No Runnable Tests | `--all` With No Runnable Targets |
| :--- | :--- | :--- | :--- | :--- |
| `test unit` | `test unit: invalid runtime options` | `unknown app "<app>"` | Print `no tests found`, exit `0` (or non-zero with `--fail-if-no-tests`) | Print `no tests found`, exit `0` (or non-zero with `--fail-if-no-tests`) |
| `test typecheck` | `typecheck: invalid runtime options` | `typecheck: unknown app "<app>"` | Print `no tests found`, exit `0` | Print `no tests found`, exit `0` |
| `test e2e` | `e2e: invalid runtime options` | `unknown module "<module>" (no package.json under <modules_path>)` | Print `no tests found`, exit `0` when module exists but has no `choysum.e2e.specs` | Exit non-zero with `e2e: no runnable modules found under <modules_path>` |

Implementation reference:
- Shared message templates: `internal/testing/semantics/messages.go`
- Contract tests: `cmd/cli_test_semantics_contract_test.go`

---

## 📜 CLA (Contributor License Agreement)

All contributors must sign the CLA before their pull request can be merged. We use `@cla-bot` to automate this process.

### 1. Individual Contributors
- Read the [Individual CLA (ICLA)](cla/icla.md).
- Submit your Pull Request. 
- **How to sign:** Once your PR is open, `@cla-bot` will prompt you. Simply reply to the PR with the phrase: **"I accept the CLA"**. The bot will automatically add your GitHub username to the allowlist.

### 2. Corporate Contributors
- If you are contributing on behalf of an employer, your organization must complete the [Corporate CLA (CCLA)](cla/ccla.md) offline.
- Email the signed document to the maintainers. 
- Once approved, a maintainer will manually add the authorized GitHub usernames to the [allowlist](cla/contributors.json).

---

## 🚀 Pull Request Checklist

1. **Focused Changes:** Keep the change focused.
2. **Tests:** Ensure `go test ./...` passes.
3. **SPDX:** Add correct SPDX headers for new files.
4. **Sign-off:** Ensure you have replied to the bot to sign the CLA.