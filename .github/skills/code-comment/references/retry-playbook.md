# Retry Playbook

Use this playbook when the skill did not auto-trigger cleanly, the request was interpreted in the wrong mode, or the scope was too broad.

## If the Skill Did Not Trigger

Retry in this order:

1. Use the slash command directly.

```text
/code-comment pkg/auth
/code-comment addons/auth/web/pages
```

2. Name the skill explicitly in natural language.

```text
Use the code-comment skill on pkg/auth comments.
Use the code-comment skill on addons/auth/web/pages Vue SFC comments.
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
/code-comment addons/auth/web/pages
/code-comment review pkg/auth without editing
/code-comment cleanup addons/web/web/router
/code-comment review addons/auth/web/pages/RoleList.vue
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
/code-comment review addons/auth/web/pages
/code-comment review pkg/auth/authenticator.go
/code-comment review addons/auth/web/pages/RoleList.vue
/code-comment review pkg/auth public interfaces
```

## If the Request Mixed Multiple Intents

Split the request into separate passes:

1. review pass
2. cleanup pass
3. optional follow-up validation request

Example:

```text
/code-comment review addons/web/web/router
/code-comment cleanup addons/web/web/router
/code-comment review addons/auth/web/pages
/code-comment cleanup addons/auth/web/pages
```

## If the Skill Still Feels Inconsistent

- Prefer explicit slash invocation over free-form prose.
- Mention the language and surface: `Go public APIs`, `TypeScript JSDoc`, `Vue SFC comments`, `router comments`, `auth package comments`.
- Ask for a narrow validation target when needed: `review pkg/auth and focus on exported Go docs only`.
- If auto-trigger quality remains poor, update `SKILL.md` `description` with the exact wording users naturally keep trying.