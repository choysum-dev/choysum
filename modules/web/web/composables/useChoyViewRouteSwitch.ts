// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, type ComputedRef } from 'vue';
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
 * Whether App/Guest shell should KeepAlive this route.
 * Explicit meta.keepAlive wins; otherwise sibling list/kanban/chart pages
 * (non-empty viewPaths) are cached by path so view-switcher navigation remounts less harshly.
 */
export function routeShouldKeepAlive(meta: unknown): boolean {
  if (!meta || typeof meta !== 'object') return false;
  const m = meta as ChoyViewRouteMeta & { keepAlive?: boolean };
  if (m.keepAlive === true) return true;
  if (m.keepAlive === false) return false;
  const paths = m.viewPaths;
  if (!paths || typeof paths !== 'object') return false;
  return Object.keys(paths).length > 0;
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
 * Bind ViewSwitcher navigation and default groupby to the current route meta.
 * Applies defaultGroups to store once during setup (sibling navigations remount views).
 */
export function useChoyViewRouteSwitch(
  options: UseChoyViewRouteSwitchOptions = {},
): UseChoyViewRouteSwitchResult {
  const route = useRoute();
  const router = useRouter();

  const active = computed(() => resolveChoyViewRouteKind(route.meta));
  const defaultGroups = computed(() => resolveChoyViewRouteDefaultGroups(route.meta));

  applyChoyViewRouteGroups(options.store, defaultGroups.value);

  function selectView(kind: string): void {
    const path = resolveChoyViewRoutePath(route.meta, kind);
    if (path) router.push(path);
  }

  return { active, defaultGroups, selectView };
}
