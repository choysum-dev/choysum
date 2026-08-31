---
name: code-comment
description: 'Clean up or review source code comments, doc comments, JSDoc, Vue SFC comments, and public API documentation in Choysum. Defaults to hybrid mode unless the user clearly asks for findings only or direct cleanup only. Use when asked to do code-comment-cleanup, code-comment-review, translate code comments to English, remove noisy comments, fill missing Go or TypeScript API comments, fill missing Vue SFC component or props/emits/slots comments, fill missing Go interface method docs or exported const-group docs, or audit comment quality in a file or directory.'
---

# Code Comment

This skill handles source-code comment maintenance for Choysum.

Use it for three closely related tasks:

- `cleanup`: apply fixes directly
- `review`: report concrete comment problems without editing
- `hybrid`: review a scoped surface, fix the concrete issues, then validate

This skill is self-contained.
Use [policy](./references/policy.md), [language checklists](./references/language-checklists.md), [review rubric](./references/review-rubric.md), [standards baseline](./references/standards-baseline.md), [recommended prompts](./references/recommended-prompts.md), and [retry playbook](./references/retry-playbook.md) as the working references.

## Decide the Mode

- Use `cleanup` when the user clearly asks for direct fixes only and does not want an initial review pass.
- Use `review` when the user asks to review, audit, inspect, compare, check, or find comment issues.
- Use `hybrid` when the user asks to review and fix in one pass, when the request mixes findings with direct cleanup, or when the request is general comment cleanup without a mode override.
- If the mode is ambiguous, default to `hybrid`. Only use `review` when the user clearly wants findings only, and only use `cleanup` when the user clearly wants direct edits without the review step.

## Scope Guardrails

- Work only on handwritten source files unless the user explicitly includes generated files or Markdown docs.
- Skip vendor, third-party, generated, lock, snapshot, and pure data files by default.
- Treat comments, doc comments, `TODO`, `FIXME`, `NOTE`, and user-facing log or error strings inside code files as English-only surfaces.
- In Vue SFCs, respect block boundaries: use the comment syntax of the current block, and use top-level HTML comments only when they add real value.
- In Vue SFCs, classify the component before reviewing it. Apply the `<script setup>` rule path to macro-based Composition API components, and apply the Options API rule path to components organized around `data`, `computed`, `methods`, `watch`, `emits`, or `expose`.
- For Choysum TypeScript and Vue script code, treat exported functions, exported classes, public fields, public methods, and named functions or methods as comment-bearing surfaces unless the file is generated or third-party.
- For Choysum model, store, service, facade, and registry classes, require a type-level doc comment plus member docs for caller-facing or schema-bearing fields and methods.
- Do not add comments to obvious code just to increase coverage.
- Do not use the "obvious code" exception to skip caller-facing API docs. In Go public packages, exported interface methods and exported enum-like constants often need documentation even when the names look short or familiar.
- Do not use the "obvious code" exception to skip named TypeScript or Vue script functions and methods when repository policy expects short stable doc comments on those declarations.
- Prefer clearer names or smaller functions over long explanatory comments when readability problems come from structure.
- When repository-specific wording is silent, follow the recognized language guidance summarized in [standards baseline](./references/standards-baseline.md).

## Cleanup Procedure

1. Resolve the requested scope to the smallest practical set of files or directories.
2. Read only enough nearby context to classify the comment problem:
   - non-English text
   - missing required public API docs
   - missing exported interface method docs
   - missing const-group or enum-member docs
   - stale or contradictory comments
   - noisy narration
   - missing constraint comments for non-obvious order, lifecycle, compatibility, or concurrency
3. Apply [policy](./references/policy.md) plus the relevant checklist in [language checklists](./references/language-checklists.md).
4. For Go public APIs, bias toward pkg.go.dev-ready docs: sentence-form comments that start with the symbol name, group comments for related const or var blocks, and per-member comments when names alone do not explain caller-facing meaning.
5. For Choysum TypeScript and Vue script code, document declarations in this order before deciding coverage is complete:
   - exported functions and exported classes
   - public fields and public methods on model, store, service, facade, and schema-bearing classes
   - named local functions and methods used for component behavior, state transitions, validation, parsing, orchestration, or side effects
6. For Vue SFCs, choose the matching Vue rule path before editing:
   - `<script setup>` and other Composition-style components: document caller-facing contract through `defineProps`, `defineEmits`, `defineModel`, `defineSlots`, `defineExpose`, and non-obvious top-level reactive bindings.
   - Options API components: document caller-facing contract through `props`, `emits`, `expose`, and only comment `data`, `computed`, `methods`, `watch`, `provide`, or `inject` when lifecycle, side effects, coupling, or public instance behavior is not obvious.
   - Under Choysum repository policy, named handlers, helper functions, and methods in the script block should still receive short doc comments even when they are not exported.
7. Make the smallest safe edit set that fixes the problem at the source.
8. After the first substantive edit, run the narrowest validation available for the touched slice:
   - editor diagnostics for the edited files
   - focused tests or type checks when the comment change is attached to public API or behavior-sensitive code
9. If validation fails, repair the same slice and rerun the same narrow check before widening scope.
10. Summarize what changed, what was validated, and any remaining risk.

## Review Procedure

1. Resolve the requested scope and stay inside it.
2. Review against [review rubric](./references/review-rubric.md) and the relevant language checklist.
3. Report only concrete comment or documentation problems.
4. Do not ask for subjective comments on obvious code.
5. Order findings by impact and keep one problem per finding.
6. Include file references and brief fix direction.
7. If no concrete issues are found, say so explicitly and mention any remaining testing or validation gap.

## Hybrid Procedure

1. Do a brief scoped review first.
2. Convert concrete findings into the smallest edit set needed.
3. Validate after the first substantive edit.
4. Continue only while the work stays within the originally requested scope.
5. Report both the issues addressed and the validation performed.

## Output Requirements

- In `cleanup` mode, summarize the edits and validations performed.
- In `review` mode, present numbered findings only; one finding per point.
- In `hybrid` mode, summarize the issues found, the edits applied, and the validations run.
- When citing rules, prefer repository-specific wording from the references instead of generic style advice.
- When repository-specific rules are too terse, use [standards baseline](./references/standards-baseline.md) to stay aligned with recognized Go, TypeScript, and Vue documentation conventions.
- When the references are not enough, stay conservative: prefer minimal cleanup and avoid inventing broader policy beyond this skill package.

## Invocation Help

- Use [recommended prompts](./references/recommended-prompts.md) when you want explicit slash-style or natural-language trigger phrases.
- Use [retry playbook](./references/retry-playbook.md) when the skill did not auto-trigger, the scope was too broad, or the first request mixed review and cleanup ambiguously.