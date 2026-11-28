# Field client validation

Conceptual **required** and the label asterisk share one source:

1. Model metadata `notNull` (FieldsGet / static meta), or
2. View `:required` (boolean / expression)

When the field is editable and visible, ChoyFieldBase automatically applies a
client `required` rule (`Required`) so blur / Form submit show the same
constraint as the asterisk. Server `notNull` remains authoritative.

**`:rules`** are additive only (patterns, custom validators, or an explicit
`required: true` with a custom message). If `rules` already include
`required: true`, the default required rule is not inserted again.

Other metadata (`size`, `selection`, `digits`, domain, unique) is not turned
into generic FieldBase rules — use control constraints, pickers, or server
errors instead.
