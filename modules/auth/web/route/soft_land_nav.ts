// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Lightweight nav candidates for permission soft-landing (no Vue page imports).
 * Sequences mirror modules/auth/web/route/routes.ts + menu/menus.ts ordering.
 */
export type SoftLandRouteCandidate = {
  path: string;
  resourceId: string;
  routeSequence: number;
  menuSequence: number;
};

const DEFAULT_SEQUENCE = Number.POSITIVE_INFINITY;

/** Navigable auth app routes used when / or /home is denied. */
export const softLandRouteCandidates: SoftLandRouteCandidate[] = [
  { path: '/auth/users', resourceId: 'auth.route.user_list', routeSequence: 10, menuSequence: 10 },
  { path: '/auth/roles', resourceId: 'auth.route.role_list', routeSequence: 10, menuSequence: 30 },
  { path: '/auth/sessions', resourceId: 'auth.route.session_list', routeSequence: 10, menuSequence: 40 },
  { path: '/auth/tokens', resourceId: 'auth.route.token_list', routeSequence: 10, menuSequence: 50 },
  { path: '/auth/record-rules', resourceId: 'auth.route.record_rule_list', routeSequence: 10, menuSequence: 35 },
  { path: '/auth/field-rules', resourceId: 'auth.route.field_rule_list', routeSequence: 10, menuSequence: 35 },
  { path: '/auth/method-accesses', resourceId: 'auth.route.method_access_list', routeSequence: 10, menuSequence: 35 },
  { path: '/auth/ui-resource-grants', resourceId: 'auth.route.ui_resource_grant_list', routeSequence: 10, menuSequence: 35 },
  { path: '/auth/users/new', resourceId: 'auth.route.user_create', routeSequence: 30, menuSequence: 10 },
  { path: '/auth/roles/new', resourceId: 'auth.route.role_create', routeSequence: 30, menuSequence: 30 },
  { path: '/auth/sessions/new', resourceId: 'auth.route.session_create', routeSequence: 30, menuSequence: 40 },
  { path: '/auth/tokens/new', resourceId: 'auth.route.token_create', routeSequence: 30, menuSequence: 50 },
  { path: '/auth/tokens/kanban', resourceId: 'auth.route.token_kanban', routeSequence: 40, menuSequence: 50 },
];

/**
 * Pick the first route path allowed by the current permission snapshot.
 */
export function pickFirstAllowedSoftLandPath(
  canRoute: (resourceId: string, state: any, ctx: any) => boolean,
  state: any,
  ctx: { activeCompanyId?: string; enabledCompanyIds?: string[] }
): string {
  const candidates = softLandRouteCandidates.filter(c => canRoute(c.resourceId, state, ctx));
  candidates.sort((a, b) => {
    if (a.routeSequence !== b.routeSequence) return a.routeSequence - b.routeSequence;
    if (a.menuSequence !== b.menuSequence) return a.menuSequence - b.menuSequence;
    const idCmp = a.resourceId.localeCompare(b.resourceId);
    if (idCmp !== 0) return idCmp;
    return a.path.localeCompare(b.path);
  });
  return candidates[0]?.path || '';
}

export { DEFAULT_SEQUENCE };
