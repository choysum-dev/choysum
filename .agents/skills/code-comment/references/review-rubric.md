# Review Rubric

Use this rubric when the request is for findings only, or when `hybrid` mode starts with a short review pass.

## Good Findings

Flag concrete issues such as:

- non-English comments, doc comments, log messages, or error strings inside code files
- missing required public API comments in Go, TypeScript, or Vue SFC contract surfaces
- missing doc comments on exported TypeScript functions or exported classes
- missing doc comments on public fields or public methods in model, store, service, facade, registry, or schema-bearing TypeScript classes
- missing short doc comments on named TypeScript or Vue script functions and methods where repository policy requires them
- exported struct, field, interface, or method semantics that remain undocumented when the language checklist requires them
- exported methods on a public Go interface that lack doc comments, even when they look getter-like
- exported const groups or exported enum-like members whose public distinction is undocumented
- missing Vue SFC contract comments where callers need help understanding component responsibility, prop semantics, emitted event payloads, model behavior, slot props, or exposed instance members
- missing model-class or schema-field comments where decorators and names alone do not fully communicate persistence, scope, relation, or contract semantics
- Vue comments or review assumptions that apply the wrong authoring-style rule path, such as demanding `defineProps` documentation from an Options API component or treating `<script setup>` bindings like `data` or `methods`
- comments that contradict the current implementation
- stale workaround comments, TODOs, or FIXMEs with no actionable context
- line-by-line narration or JSDoc that simply repeats type information
- missing local comments where order, lifecycle, compatibility, concurrency, or security constraints are not otherwise visible
- tool-hostile JSDoc usage in TypeScript, such as dumping long prose into the summary or repeating TS type declarations with JS-only tags
- Vue block comments that use the wrong syntax for the SFC block, or comments that merely restate obvious template or prop declarations

## Do Not Flag

- subjective requests for more comments on obvious code
- formatter-only or whitespace-only style preferences
- missing comments in generated or third-party code unless the user explicitly scoped those files in
- Markdown documentation issues when the request is limited to code files
- broad architectural documentation gaps when the local code comments already meet the requested scope
- missing `defineProps` or `defineEmits` comments on an Options API component, or missing `data` or `methods` comments on a `<script setup>` component, when the wrong Vue checklist was applied

## Review Output

- Findings first, ordered by impact
- One problem per finding
- Include the file path and a short explanation of why it violates the policy
- Add a brief fix direction, not a full redesign unless the comment problem is caused by structure
- If no findings exist, state that clearly and mention any residual validation gap