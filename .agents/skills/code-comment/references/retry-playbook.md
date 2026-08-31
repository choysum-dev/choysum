# Retry Playbook

Use this playbook when the skill did not auto-trigger cleanly, the request was interpreted in the wrong mode, or the scope was too broad.

## If the Skill Did Not Trigger

Retry in this order:

1. Use the slash command when the host supports skill slash commands (for example Cursor or Copilot agent mode).

```text
/code-comment pkg/auth
/code-comment modules/auth/web/pages
```

2. Name the skill explicitly in natural language (portable fallback on any host).

```text
Use the code-comment skill on pkg/auth comments.
Use the code-comment skill on modules/auth/web/pages Vue SFC comments.
```

3. Reuse the discovery phrases from the skill description.

```text
Use the code-comment skill to do code-comment-review for pkg/auth.
Use the code-comment skill to translate code comments to English in router.
Use the code-comment skill to review Vue SFC comments in RoleList.vue.
```

## If the Wrong Mode Was Chosen

- Add `cleanup` when you want direct edits without the review step.
- Add `review` plus `without editing` when you want findings only.
- Add `hybrid` when you want to override back to the default short review plus fixes flow.

If you omit the mode entirely, the skill should default to `hybrid`.

Examples:

```text
/code-comment pkg/auth
/code-comment modules/auth/web/pages
/code-comment review pkg/auth without editing
/code-comment cleanup modules/web/web/router
/code-comment review modules/auth/web/pages/RoleList.vue
/code-comment hybrid pkg/auth public APIs
```

## If the Scope Was Too Broad

Reduce the target in this order:

1. directory -> subdirectory
2. subdirectory -> file set
3. file set -> single file or single public API surface

Examples:

```text
/code-comment review pkg/auth
/code-comment review modules/auth/web/pages
/code-comment review pkg/auth/authenticator.go
/code-comment review modules/auth/web/pages/RoleList.vue
/code-comment review pkg/auth public interfaces
```

## If the Request Mixed Multiple Intents

Default to `hybrid`: one pass that reviews the scoped surface, applies concrete fixes, then validates.

Use separate review and cleanup passes only when:

- the host cannot execute `hybrid` in one turn, or
- the user explicitly asks for findings first and edits later.

Example (hybrid — default):

```text
Use the code-comment skill in hybrid mode on modules/web/web/router.
/code-comment hybrid modules/web/web/router
```

Example (separate passes — when requested or host-limited):

```text
/code-comment review modules/web/web/router without editing
/code-comment cleanup modules/web/web/router
```

## If the Skill Still Feels Inconsistent

- Prefer explicit natural-language skill naming; slash invocation (for example `/code-comment`) works only on hosts that expose skill slash commands.
- Mention the language and surface: `Go public APIs`, `TypeScript JSDoc`, `Vue SFC comments`, `router comments`, `auth package comments`.
- Ask for a narrow validation target when needed: `review pkg/auth and focus on exported Go docs only`.
- If auto-trigger quality remains poor, update `SKILL.md` `description` with the exact wording users naturally keep trying.