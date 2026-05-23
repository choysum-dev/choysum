# Language Checklists

Use these checklists to decide whether a file needs comment work and what kind.

## Shared Precheck

- The file is handwritten source, not generated or third-party code.
- All retained comments and user-facing code-file text are English.
- Retained comments explain intent, contract, invariants, boundaries, or traps.
- Noise comments are removed before new comments are added.

## Go Checklist

- Public or owner packages that need boundary context have one package comment placed directly above `package`.
- Exported functions, methods, types, interfaces, variables, ungrouped constants, and exported const or var groups have Go-style doc comments whose first sentence starts with the symbol name or clearly names the group.
- Exported structs cover exported field semantics either through the type doc or field comments.
- Exported interfaces explain the overall contract.
- In public or cross-package Go interfaces, every exported method is documented unless the interface comment already covers a truly trivial method with no caller or implementer ambiguity.
- Getter-like interface methods are not exempt when callers need to know where data comes from, whether metadata can be mutated, what validity means, or how identifiers relate to tokens or users.
- Exported enum-like const members in token, state, mode, protocol, or error-code groups have short item comments when the member meaning is part of the public contract.
- Type comments explain zero-value or concurrency guarantees when needed.
- Group docs exist for exported constant or error-code groups whose names do not fully explain the shared meaning.
- Internal block comments appear only where order, lifecycle, compatibility, security, or concurrency is non-obvious.
- Go doc formatting does not accidentally create malformed lists, fake code blocks, or broken links.

## TypeScript and JavaScript Checklist

- Cross-file public exports with contract meaning have JSDoc.
- Exported functions and exported classes have JSDoc even when the summary is short.
- Public fields and public methods on model, store, service, facade, registry, or schema-bearing classes have JSDoc.
- Named functions and methods in `.ts`, `.tsx`, and Vue script blocks have at least a short summary comment under repository policy.
- JSDoc keeps a short summary first; longer behavior notes go in `@remarks` when needed.
- Standard tags such as `@param`, `@returns`, `@example`, `@deprecated`, `@see`, and `{@link ...}` are used only when they add contract value.
- `.ts` and `.tsx` files do not repeat existing TypeScript declarations with `@type` or `@typedef` unless the file is actually relying on JS-style tooling behavior.
- Fields, props, and options are commented only when units, defaults, legal ranges, compatibility meaning, or wire formats are not obvious.
- Decorator-based model or schema classes include a type doc comment plus member docs for persisted, relational, derived, validation, or scope-carrying fields.
- Internal comments exist only for hidden side effects, fallback paths, cache rules, ordering rules, or data-shape constraints.

## Vue SFC Shared Checklist

- `.vue` files follow the SFC block comment rules: use the syntax of the current block, and use top-level HTML comments only when they add durable component context.
- Component-level comments are minimal and explain responsibility, boundary, lifecycle, or integration meaning only when the filename and local structure are not enough.
- Template comments exist only for non-obvious rendering constraints, fallback behavior, SSR or hydration constraints, accessibility reasoning, or ordering traps.
- Style block comments exist only for scoping strategy, design-token coupling, browser workarounds, or compatibility constraints that are not obvious from selectors and declarations.
- Vue comments do not restate information that is already encoded in prop definitions, TypeScript types, validators, option declarations, or obvious template structure.

### Vue `<script setup>` Checklist

- In `<script setup>`, do not comment every top-level binding just because it is visible to the template.
- Named functions in the script block, including event handlers and orchestration helpers, carry short doc comments under repository policy.
- `defineProps`, runtime `props`, and related prop types are commented only when names, types, defaults, validators, or required flags do not fully explain contract, units, legal values, compatibility meaning, or ownership.
- `defineEmits`, emitted event names, and payload semantics are commented when callers would not understand the event contract from the name and type alone.
- `defineModel` declarations are commented when sync behavior, defaulting, modifiers, transforms, or parent-child expectations are non-obvious.
- `defineSlots` declarations or slot-related comments explain slot props and behavioral expectations when slot names or types alone are not enough for callers.
- `defineExpose` members are commented when the exposed instance surface is part of the component's supported contract.
- If a normal `<script>` block exists alongside `<script setup>`, comment only module-scope side effects or options that cannot already be expressed cleanly inside `<script setup>`.

### Vue Options API Checklist

- `props` declarations are commented only when names, types, defaults, validators, or required flags do not fully explain contract, units, legal values, compatibility meaning, or ownership.
- `emits` option declarations and payload validators are commented when callers would not understand the event contract from the event name and validator shape alone.
- `data` state comments are rare and explain units, ownership, persistence, or lifecycle coupling only when names and nearby usage are insufficient.
- `computed` properties and `methods` do not get narration, but named methods still carry short doc comments under repository policy. Expand beyond the summary only for caller-visible behavior, side effects, caching assumptions, or ordering constraints that are not obvious from the option name and body.
- `watch` entries are commented only when watched source choice, `deep` or `immediate` or `flush` behavior, cleanup, or cross-component coupling is not obvious.
- `expose`, `provide`, and `inject` options are commented when public instance surface or dependency boundaries are part of the supported contract.
- `mixins` and `extends` need short constraint comments only when merge order, inheritance assumptions, or legacy boundaries would otherwise be easy to misread.

## Test Checklist

- Test names already carry the main behavior expectation.
- Comments only explain scenario motive, fixture reason, historical regression background, or environment limits.
- Arrange/Act/Assert narration is removed.
- Temporary skips or TODO markers explain the unlock condition.