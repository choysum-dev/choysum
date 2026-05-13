# CLA Documents

This directory contains the canonical Contributor License Agreements for Choysum.

## 🚀 Signing Workflow

- **Individuals**: Submit your PR and reply to the `@cla-bot` comment with the exact phrase: **"I accept the CLA"**. The bot will automatically update the allowlist.
- **Corporations**: Email a signed [ccla.md](ccla.md) to the maintainers. We will manually add your team to the allowlist after verification.

## 🛠️ Internal Notes

- **Allowlist**: [contributors.json](contributors.json) is the source of truth for `@cla-bot`.
- **Security**: **NEVER** upload signed CCLA PDFs to this repository. All signed documents must be kept in secure, private storage.
- **Bot Config**: The root [../.clabot](../.clabot) file defines the bot's interaction logic and the required signing phrase.
- **Policy Changes**: If the CLA terms are updated, ensure the version numbers in [icla.md](icla.md) and [ccla.md](ccla.md) are incremented and a project-wide announcement is made.