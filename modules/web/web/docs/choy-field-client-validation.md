# Field client validation

Conceptual **required** and the label asterisk share one source:

1. Model metadata `notNull` (FieldsGet / static meta), or
2. View `:required` (boolean / expression)

When the field is editable and visible, ChoyFieldBase automatically applies a
client `required` check so blur / Form submit stay aligned with the label
asterisk. **Auto-required** (meta.notNull / `:required` without an explicit
`rules` required) fails validation with Form toast + invalid chrome (red
label / control) and does **not** paint inline「必填」copy. The asterisk is
edit/create only — display mode omits the glyph but still reserves its width.
Server `notNull` remains authoritative.

**`:rules`** are additive. Supported checks are `required: true` (with an
optional custom message) and custom `validator` functions (e.g. wrap a regex
inside `validator` — a bare `pattern` property is not evaluated). Explicit
`required: true` in `rules` still paints its message. If `rules` already
include `required: true`, the default required rule is not inserted again.

Other metadata (`size`, `selection`, `digits`, domain, unique) is not turned
into generic FieldBase rules — use control constraints, pickers, or server
errors instead.
