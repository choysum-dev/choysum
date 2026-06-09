# Recommended Prompts

Use these short prompts when you want to trigger the skill quickly in day-to-day VS Code chat.

## Slash Invocations

Prefer slash invocations when you want deterministic skill loading.

```text
/code-comment pkg/auth
/code-comment modules/web/web/router
/code-comment modules/auth/web/pages
/code-comment review modules/auth/web/pages/RoleList.vue
/code-comment review pkg/auth
/code-comment hybrid pkg/auth
/code-comment review modules/web/web/router
/code-comment hybrid modules/web/web/router
```

## Natural-Language Triggers

```text
Use code-comment on pkg/auth.
Use code-comment on modules/web/web/router.
Use code-comment on modules/auth/web/pages.
Use code-comment review on modules/auth/web/pages/RoleList.vue.
Use code-comment hybrid on modules/web/web/components/view/OFormView.vue Vue SFC comments.
Use code-comment review on pkg/auth without editing.
Use code-comment hybrid on pkg/auth public APIs.
Use code-comment review on modules/web/web/router.
Use code-comment hybrid on modules/web/web/router.
```

## Copy Tips

- Keep the scope to one file, directory, or API surface.
- Omit the mode when you want the default `hybrid` behavior.
- Include `cleanup`, `review`, or `hybrid` only when you want to override the default.
- Add `without editing` when you want findings only.
- Add `public APIs`, `comments`, `JSDoc`, or `Vue SFC` when you want a tighter target.