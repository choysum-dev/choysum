<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

# Selection field values

`type: 'selection'` options are **symbolic keys** plus display labels — not a free-form literal dictionary.

## Contract

| Piece | Rule |
| --- | --- |
| `value` | Non-empty string **without** leading or trailing whitespace (including Unicode spaces such as NBSP). Declaration / merge **reject** dirty keys; they are not silently trimmed. |
| `label` | Human-facing text (string or `_lt(...)`). |
| Write validation | Exact match against the merged option list (no second trim on input). |

Meaningful whitespace literals (space, NBSP) and empty separators are **out of scope** for model selection. Prefer:

- `varchar` (store the literal) + form `ChoySelectionField :selection` for UI picks, or
- tokenized selection values (`space` / `nbsp` / `none`) with decode at the format boundary

Do not add a second “exact literal” selection mode unless multiple real consumers need it.
