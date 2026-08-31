# Standards Baseline

Use this summary when repository-specific rules need a public, language-level baseline.

## Go Baseline

- Follow the official Go doc comment conventions from `go.dev/doc/comment`.
- Every exported top-level name should have a doc comment.
- Doc comments are complete sentences that start with the declared name.
- Type comments explain what the type represents or provides, including zero-value or concurrency guarantees when those matter to callers.
- Exported structs explain exported field meaning either in the type doc or with field comments.
- Function and method comments explain what the operation returns or does, including caller-visible special cases.
- Boolean-returning docs should usually use the phrase `reports whether`.
- Related const or var blocks may use one group comment plus short item comments.
- For exported enum-like, token-like, status-like, mode-like, or protocol-facing constants, add item comments when callers need the distinction.
- Public interfaces should document the contract callers and implementers rely on. If the method set is the API, document the methods, not only the interface name.

## TypeScript Baseline

- Follow tool-friendly TSDoc or JSDoc conventions that multiple tools can parse consistently.
- Keep a short summary first; move longer explanation to `@remarks` when needed.
- Use `@param`, `@returns`, `@deprecated`, `@see`, and `{@link ...}` only when they add contract, lifecycle, or navigation value.
- In `.ts` and `.tsx`, do not repeat existing TypeScript type information with JS-only type tags unless tooling explicitly requires it.
- Document public exports when callers need contract, return semantics, side effects, lifecycle behavior, wire format, compatibility notes, or deprecation guidance.
- Prefer comment forms that remain stable across editors, API generators, and lint or extraction tools.

## Vue SFC Baseline

- Follow the official Vue SFC syntax specification for block structure and comment syntax.
- Treat `.vue` files as multi-block source files: template, script, style, and optional custom blocks may each need different comment syntax and different thresholds for adding comments.
- Vue components can be authored in Options API or Composition API. Keep those rule paths separate when reviewing comment quality instead of mixing their contract surfaces.
- Template comments should be rare and should explain non-obvious rendering constraints, fallback behavior, SSR or hydration assumptions, accessibility reasoning, or ordering traps.
- Style comments in SFCs should be rare and should explain scoping strategy, design-token coupling, browser workarounds, or compatibility constraints.

### Vue `<script setup>` / Composition-Style Baseline

- In SFCs, Composition API is typically used with `<script setup>`, and Vue recommends that form for SFC plus Composition API code.
- Comments should focus on caller-facing contract and non-obvious constraints, not on every top-level binding exposed to the template.
- For component contracts, prefer the first-class Vue surfaces that callers actually use: `defineProps`, `defineEmits`, `defineModel`, slots, and exposed instance members.
- Use comments to clarify prop semantics only when names, types, defaults, validators, or required flags do not fully explain units, legal values, ownership, lifecycle, or compatibility meaning.
- Use comments to clarify event payload meaning, slot props, and `defineExpose` surfaces when the contract is not obvious from names and types alone.
- Respect Vue's detailed prop-definition guidance: comments supplement explicit prop typing and defaults instead of replacing them.

### Vue Options API Baseline

- Options API is centered on a component instance and organizes logic under option groups such as `data`, `computed`, `methods`, lifecycle hooks, `props`, `emits`, and `expose`.
- In Options API components, prioritize comments on public contract surfaces first: `props`, `emits`, `expose`, and non-obvious dependency or lifecycle boundaries.
- `data`, `computed`, and `methods` rarely need comments unless units, side effects, ordering, or cache or derived-state assumptions are not obvious from the option name and body.
- `watch` comments should explain non-obvious source choice, deep traversal, eager timing, flush timing, cleanup, or cross-component coupling when those behaviors matter.
- Because `this` already conveys the component-instance model, comments should explain contract or coupling rather than restating that an option reads or mutates instance state.
- `mixins` remain supported but are no longer the preferred reuse mechanism in Vue 3, and `extends` is designed for Options API inheritance. When these surfaces remain in code, comment only merge order, inheritance constraints, or migration boundaries that maintainers need.

## Sources

- Go official doc comments: https://go.dev/doc/comment
- Google Go style commentary and interfaces: https://google.github.io/styleguide/go/decisions#doc-comments
- Google Go comment sentences: https://google.github.io/styleguide/go/decisions#comment-sentences
- Google Go interface guidance: https://google.github.io/styleguide/go/decisions#interfaces
- TSDoc overview: https://tsdoc.org/
- TypeScript JSDoc reference: https://www.typescriptlang.org/docs/handbook/jsdoc-supported-types.html
- Vue API styles overview: https://vuejs.org/guide/introduction#api-styles
- Vue SFC syntax specification: https://vuejs.org/api/sfc-spec
- Vue `<script setup>` reference: https://vuejs.org/api/sfc-script-setup
- Vue Options: State reference: https://vuejs.org/api/options-state
- Vue Options: Composition reference: https://vuejs.org/api/options-composition
- Vue style guide essential rules: https://vuejs.org/style-guide/rules-essential.html