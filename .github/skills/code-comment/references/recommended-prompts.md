# Recommended Prompts

Use these short prompts when you want to trigger the skill quickly in day-to-day VS Code chat.

## Slash Invocations

Prefer slash invocations when you want deterministic skill loading.

```text
/code-comment pkg/auth
/code-comment addons/web/web/router
/code-comment addons/auth/web/pages
/code-comment review addons/auth/web/pages/RoleList.vue
/code-comment review pkg/auth
/code-comment hybrid pkg/auth
/code-comment review addons/web/web/router
/code-comment hybrid addons/web/web/router
```

## Natural-Language Triggers

```text
Use code-comment on pkg/auth.
Use code-comment on addons/web/web/router.
Use code-comment on addons/auth/web/pages.
Use code-comment review on addons/auth/web/pages/RoleList.vue.
Use code-comment hybrid on addons/web/web/components/view/OFormView.vue Vue SFC comments.
Use code-comment review on pkg/auth without editing.
Use code-comment hybrid on pkg/auth public APIs.
Use code-comment review on addons/web/web/router.
Use code-comment hybrid on addons/web/web/router.
```

## Copy Tips

- Keep the scope to one file, directory, or API surface.
- Omit the mode when you want the default `hybrid` behavior.
- Include `cleanup`, `review`, or `hybrid` only when you want to override the default.
- Add `without editing` when you want findings only.
- Add `public APIs`, `comments`, `JSDoc`, or `Vue SFC` when you want a tighter target.