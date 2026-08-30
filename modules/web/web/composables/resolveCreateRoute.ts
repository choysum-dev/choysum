// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, toValue, type ComputedRef, type MaybeRefOrGetter } from 'vue';
import { useRoute, useRouter, type RouteLocationRaw, type Router } from 'vue-router';

const SURFACE_SUFFIX = /(List|Detail|Kanban)$/;

/**
 * Map a screen route name to its sibling Create route name.
 * `PartnerList` / `PartnerDetail` / `TokenKanban` → `PartnerCreate` / `TokenCreate`.
 * Already-`Create` names are returned unchanged so form "New" works on the create screen.
 */
export function deriveCreateRouteName(routeName: unknown): string | undefined {
  if (typeof routeName !== 'string' || !routeName) {
    return undefined;
  }
  if (routeName.endsWith('Create')) {
    return routeName;
  }
  if (!SURFACE_SUFFIX.test(routeName)) {
    return undefined;
  }
  const createName = routeName.replace(SURFACE_SUFFIX, 'Create');
  return createName;
}

/**
 * Resolve `{ name: XxxCreate }` when that route exists; otherwise undefined.
 */
export function resolveCreateRouteLocation(router: Router, routeName: unknown): RouteLocationRaw | undefined {
  const createName = deriveCreateRouteName(routeName);
  if (!createName) {
    return undefined;
  }
  try {
    const resolved = router.resolve({ name: createName });
    if (!resolved.matched.length || resolved.name !== createName) {
      return undefined;
    }
    return { name: createName };
  } catch {
    return undefined;
  }
}

/**
 * Prefer an explicit createAction prop; when omitted, derive Create from the current route name.
 * Pass `''` to disable both the prop target and route-name fallback.
 * Set `enabled: false` to skip route-name fallback (e.g. embedded forms) while still honoring an explicit prop.
 */
export function useResolvedCreateAction(
  propCreateAction: MaybeRefOrGetter<string | RouteLocationRaw | null | undefined>,
  options?: { enabled?: MaybeRefOrGetter<boolean> }
): ComputedRef<string | RouteLocationRaw | undefined> {
  const router = useRouter();
  const route = useRoute();
  return computed(() => {
    const fromProp = toValue(propCreateAction);
    if (fromProp === '') {
      return undefined;
    }
    if (fromProp !== undefined && fromProp !== null) {
      return fromProp;
    }
    const enabled = options?.enabled === undefined ? true : toValue(options.enabled);
    if (!enabled) {
      return undefined;
    }
    return resolveCreateRouteLocation(router, route.name);
  });
}
