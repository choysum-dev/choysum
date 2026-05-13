# Contributing Guide

## License Scope

- Default (outside addons/): LGPL-3.0-or-later
- addons/: Apache-2.0
- Canonical files: [LICENSE](LICENSE) and [addons/LICENSE](addons/LICENSE)

## SPDX Headers

All new source files should include an SPDX line at the top.

- Outside addons/: SPDX-License-Identifier: LGPL-3.0-or-later
- Inside addons/: SPDX-License-Identifier: Apache-2.0

## Boundary Rules

- Do not copy or vendor core source code into addons/.
- Keep code in the directory that matches its intended license.
- Check third-party license compatibility before adding dependencies.

## Pull Request Checklist

- Keep the change focused.
- Add correct SPDX headers for new files.
- Explain license impact in the PR description when relevant.

## CLA (Optional)

If CLA is enabled in the future, prefer a license-based CLA model.
