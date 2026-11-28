// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, getCurrentInstance, onActivated, watch, type ComputedRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';

/**
 * Route meta fields for multi-view management pages (list / kanban / chart / …).
 * Declared on sibling routes; consumed by useChoyViewRouteSwitch.
 */
export type ChoyViewRouteMeta = {
  /** Active view kind for this route (e.g. list, kanban, chart). */
  viewKind?: string;
  /** Default store appliedGroups for this view. */
  defaultGroups?: string[];
  /** Absolute paths keyed by view kind for ViewSwitcher navigation. */
  viewPaths?: Record<string, string>;
};

export type UseChoyViewRouteSwitchOptions = {
  /** Shared model store whose queryState.appliedGroups should follow route meta. */
  store?: unknown;
};

export type UseChoyViewRouteSwitchResult = {
  active: ComputedRef<string>;
  defaultGroups: ComputedRef<string[]>;
  selectView: (kind: string) => void;
};

/** Normalize route paths for equality (trim trailing slash except root). */
export function normalizeChoyViewRoutePath(path: string): string {
  const trimmed = String(path || '').trim();
  if (!trimmed) return '';
  if (trimmed === '/') return '/';
  // Avoid /\/+$/ on attacker-controlled paths (CodeQL js/polynomial-redos).
  let end = trimmed.length;
  while (end > 1 && trimmed.charCodeAt(end - 1) === 47 /* '/' */) {
    end -= 1;
  }
  return end === trimmed.length ? trimmed : trimmed.slice(0, end);
}

/** Read viewKind from route meta; blank / missing falls back to list. */
export function resolveChoyViewRouteKind(meta: unknown, fallback = 'list'): string {
  if (!meta || typeof meta !== 'object') return fallback;
  const kind = (meta as ChoyViewRouteMeta).viewKind;
  if (typeof kind !== 'string') return fallback;
  const trimmed = kind.trim();
  return trimmed !== '' ? trimmed : fallback;
}

/** Read defaultGroups from route meta (non-arrays → []). */
export function resolveChoyViewRouteDefaultGroups(meta: unknown): string[] {
  if (!meta || typeof meta !== 'object') return [];
  const groups = (meta as ChoyViewRouteMeta).defaultGroups;
  if (!Array.isArray(groups)) return [];
  return groups.filter((g): g is string => typeof g === 'string');
}

/** Resolve a sibling path for kind from meta.viewPaths. */
export function resolveChoyViewRoutePath(meta: unknown, kind: string): string {
  if (!meta || typeof meta !== 'object') return '';
  const paths = (meta as ChoyViewRouteMeta).viewPaths;
  if (!paths || typeof paths !== 'object') return '';
  const raw = paths[kind];
  return typeof raw === 'string' ? raw.trim() : '';
}

/**
 * Whether App shell should KeepAlive this route (Guest never KeepAlive).
 * Explicit meta.keepAlive wins (defineRoute already defaults true for
 * collection route ids). When omitted at runtime, pageKind === 'collection'
 * is cached. viewPaths is ViewSwitcher-only and does not imply KeepAlive.
 * Instance lifetime is further scoped by the shell keepAliveScopeKey
 * (company + sticky activeMenuId).
 */
export function routeShouldKeepAlive(meta: unknown): boolean {
  if (!meta || typeof meta !== 'object') return false;
  const m = meta as ChoyViewRouteMeta & {
    keepAlive?: boolean;
    pageKind?: string;
  };
  if (m.keepAlive === true) return true;
  if (m.keepAlive === false) return false;
  return typeof m.pageKind === 'string' && m.pageKind.trim() === 'collection';
}

/**
 * Write appliedGroups onto a model store queryState.
 * Creates state / queryState when missing; ignores non-object stores.
 */
export function applyChoyViewRouteGroups(store: unknown, groups: readonly string[]): void {
  if (!store || typeof store !== 'object') return;
  const root = store as { state?: Record<string, unknown> };
  const state = (root.state || (root.state = {})) as Record<string, unknown>;
  const qs = (state.queryState || (state.queryState = {})) as Record<string, unknown>;
  qs.appliedGroups = [...groups];
}

/**
 * Clear Search pagination offset on a shared model store (keep limit).
 * Used when ViewSwitcher changes viewKind so List page N does not bleed into Kanban lanes.
 */
export function resetChoyViewRoutePaginationOffset(store: unknown): void {
  if (!store || typeof store !== 'object') return;
  const root = store as { state?: Record<string, unknown> };
  const state = root.state;
  if (!state || typeof state !== 'object') return;
  const qs = state.queryState as Record<string, unknown> | undefined;
  if (!qs || typeof qs !== 'object') return;
  const pagination = qs.pagination as { limit?: unknown; offset?: unknown } | undefined;
  if (!pagination || typeof pagination !== 'object') return;
  const offset = Number(pagination.offset);
  if (!Number.isFinite(offset) || offset === 0) return;
  const limitRaw = Number(pagination.limit);
  const limit = Number.isFinite(limitRaw) && limitRaw > 0 ? limitRaw : 20;
  qs.pagination = { ...pagination, limit, offset: 0 };
}

/** Last viewKind applied per shared store (survives sibling KeepAlive remounts). */
const lastViewKindByStore = new WeakMap<object, string>();

/**
 * Bind ViewSwitcher navigation and default groupby to the current route meta.
 * Applies defaultGroups on setup and again when a KeepAlive view is reactivated
 * so sibling views sharing a store do not keep another view's grouping.
 * When viewKind changes for the same store, pagination offset is cleared (keep limit).
 */
export function useChoyViewRouteSwitch(
  options: UseChoyViewRouteSwitchOptions = {},
): UseChoyViewRouteSwitchResult {
  const route = useRoute();
  const router = useRouter();

  const active = computed(() => resolveChoyViewRouteKind(route.meta));
  const defaultGroups = computed(() => resolveChoyViewRouteDefaultGroups(route.meta));

  function applyGroups(): void {
    const kind = active.value;
    const store = options.store;
    if (store && typeof store === 'object') {
      const prev = lastViewKindByStore.get(store);
      if (prev !== undefined && prev !== kind) {
        resetChoyViewRoutePaginationOffset(store);
      }
      lastViewKindByStore.set(store, kind);
    }
    applyChoyViewRouteGroups(store, defaultGroups.value);
  }

  applyGroups();
  watch(
    defaultGroups,
    (next, prev) => {
      // Fresh but equal arrays (query-only navigation) must not reset selection.
      if (
        prev &&
        next.length === prev.length &&
        next.every((g, i) => g === prev[i])
      ) {
        return;
      }
      applyGroups();
    },
    { deep: true },
  );
  if (getCurrentInstance()) {
    onActivated(applyGroups);
  }

  function selectView(kind: string): void {
    const path = resolveChoyViewRoutePath(route.meta, kind);
    if (path) router.push(path);
  }

  return { active, defaultGroups, selectView };
}
