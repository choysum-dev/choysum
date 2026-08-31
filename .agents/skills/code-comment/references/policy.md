# Comment Policy

Use this summary when you need the repository policy without loading the full guide.

## Global Rules

- Apply the policy to handwritten source files.
- Comments, doc comments, `TODO`, `FIXME`, `NOTE`, log messages, and error messages inside code files must be English.
- Comments exist to explain intent, contract, invariants, boundaries, and traps.
- Do not use comments to narrate obvious code.
- Prefer short, stable comments. If the background takes many lines, move it to Markdown and leave only a code-local anchor comment.

## Required Comment Surfaces

- Go exported packages need package comments when they expose public responsibility, entrypoint behavior, or easy-to-misread boundaries.
- Go exported top-level names need doc comments written for pkg.go.dev and IDE readers.
- Go exported structs must cover exported field semantics either in the type comment or in field comments.
- Go exported interfaces must document the overall contract.
- Go exported methods on exported interfaces in public or cross-package APIs should be documented as part of the contract, including getter-like methods when callers need to understand source, ownership, validity, mutability, or lifetime semantics.
- Go exported const or var groups need a group comment when the declarations form a related public set.
- Exported enum-like, token-like, status-like, mode-like, or error-code members need short item comments when callers must distinguish their semantic role, wire value, lifecycle meaning, or protocol meaning.
- TypeScript cross-file public exports need JSDoc when they carry contract, return, side-effect, or lifecycle meaning.
- In Choysum TypeScript code, exported functions and exported classes are required doc-comment surfaces.
- In Choysum TypeScript code, public fields and public methods on model, store, service, facade, registry, or schema-bearing classes are required doc-comment surfaces.
- In Choysum TypeScript and Vue script code, named functions and methods should carry short stable doc comments under repository policy, even when they are file-local.
- Decorator-based model or schema classes need a type-level doc comment plus field comments for persisted, relational, derived, or contract-bearing fields whose semantics are not fully captured by the decorator arguments alone.
- Vue SFC components need comment cleanup at the component-contract level, not blanket narration: preserve or add comments only when callers need help with component responsibility, props, emits, model behavior, slot props, exposed instance members, async setup constraints, or integration boundaries.
- In Vue SFCs, use block-appropriate comment syntax. Top-level comments use HTML syntax per the SFC spec, while template, script, and style blocks use the syntax of their own language.
- For Vue SFCs, choose one authoring-style path before judging comment coverage. `<script setup>` or other Composition-style components are reviewed through macros and top-level reactive contract surfaces, while Options API components are reviewed through instance option groups such as `props`, `emits`, `data`, `computed`, `methods`, `watch`, `provide`, `inject`, and `expose`.
- Non-obvious control semantics need local comments near the code.

## Standards Alignment

- Default to the recognized language conventions summarized in [standards baseline](./standards-baseline.md).
- For Go, that means exported API docs should read cleanly in pkg.go.dev, start with the symbol name, and focus on caller-visible contract rather than implementation trivia.
- For TypeScript, that means tool-friendly TSDoc or JSDoc with a short summary first and tags only when they add contract value.
- For Vue SFCs, that means following the official SFC syntax rules, treating `defineProps`, `defineEmits`, `defineModel`, `defineSlots`, and `defineExpose` as the main contract surfaces, and avoiding comments that just restate obvious template or reactive code.
- Vue's own API-styles guidance distinguishes Options API from Composition API. The skill should preserve that split instead of mixing `<script setup>` expectations into Options API components or vice versa.

## Recommended but Minimal

- Entry or registration files with side effects may need a short responsibility note.
- Heavy fixtures or regression-driven tests may need a short scenario note.
- Fields or options may need comments when units, defaults, valid ranges, wire formats, or compatibility meaning are not obvious.
- Vue components may need a short contract note when public props, events, slot props, or exposed members are otherwise easy to misread.

## Default No-Comment Areas

- Obvious assignments, branches, loops, and return assembly
- Thin forwarding wrappers
- Declarations whose meaning is already clear from naming and typing
- Route tables, menu tables, registries, and similar declarative lists unless hidden constraints exist
- Obvious template markup or reactive bindings whose behavior is already clear from the SFC structure

The previous no-comment defaults do not override the repository requirement to document named TypeScript or Vue script functions and methods, or public fields and methods on model and store-like classes.

## Fixed Marker Policy

- `TODO`: confirmed work that is not finished yet
- `FIXME`: known broken or incomplete behavior
- `NOTE`: stable maintainer context that must not be missed

Markers must include useful context such as reason, issue, owner, removal condition, or time window.