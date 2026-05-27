# Security Policy

## Supported Versions

Security fixes are currently provided for:

| Version | Supported |
| --- | --- |
| main branch (latest commit) | Yes |
| release tags / older commits | Best effort only |

## Reporting a Vulnerability

Please do not open public issues for security vulnerabilities.

Use one of the following private channels:

1. GitHub Private Vulnerability Reporting (preferred):
   - Open the repository Security tab.
   - Choose "Report a vulnerability" to submit a private report.
2. Security advisory workflow:
   - If you already have maintainer access, open a draft advisory in the repository advisories section.

## What to Include in a Report

Please include as much detail as possible:

- A clear description of the issue and impact.
- Affected components, modules, endpoints, or files.
- Reproduction steps or a proof-of-concept.
- Any logs, traces, screenshots, or stack traces that help reproduce the issue.
- Suggested remediation if available.

## Response Targets

Maintainer response targets are:

- Initial acknowledgment: within 72 hours.
- Triage and severity assessment: within 7 days.
- Mitigation or fix plan: as soon as practical based on severity and exploitability.

These targets are goals, not strict guarantees.

## Disclosure Process

- We will validate and triage the report privately.
- We may ask for additional details or test cases.
- Once a fix is ready, we will coordinate disclosure timing.
- Public disclosure should happen only after a fix (or mitigation guidance) is available.

## Safe Testing Guidelines

When testing this project for vulnerabilities:

- Do not access or modify data that does not belong to you.
- Do not perform denial-of-service, destructive load, or social-engineering attacks.
- Keep testing scoped to this repository and systems you are explicitly authorized to test.
- Avoid sharing exploit details publicly before coordinated disclosure.

Thank you for helping keep this project and its users safe.
