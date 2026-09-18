// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Authz Search / grant row shapes shared across auth service helpers.
 *
 * Kept in `auth` (not `core`): these are meta/auth domain projections, not ORM
 * primitives. Prefer `Projected<meta.MetaUiResource, F>` once HC7 call sites can
 * carry literal `fields` without hand-rolled bags; until then keep one SSOT here.
 *
 * Bags use PascalCase field names only (Id-only refs).
 */

/**
 * `meta.MetaUiResource.Search` projection used by method-access and permission-state UI.
 * Field set is the union of those call sites (extra keys stay optional).
 */
export type MetaUiResourceAuthzRow = {
  Id?: unknown;
  Name?: unknown;
  Type?: unknown;
  ParentId?: unknown;
  MetaApplicationId?: unknown;
  Requires?: unknown;
};

/**
 * `meta.MetaUiResourceMenuRoute.Search` projection for menu↔route edges.
 */
export type MetaUiResourceMenuRouteRow = {
  MenuUiResourceId?: unknown;
  RouteUiResourceId?: unknown;
};

/**
 * `meta.MetaUiResourceRouteAction.Search` projection for route↔action edges.
 */
export type MetaUiResourceRouteActionRow = {
  RouteUiResourceId?: unknown;
  ActionUiResourceId?: unknown;
};

/**
 * `auth.RoleUiResource` grant / access row (Mode + resource or application scope).
 * Not a MetaUiResource bag — do not rename back to UiResourceRow.
 */
export type RoleUiResourceGrantRow = Record<string, unknown> & {
  Id?: string;
  Mode?: string;
  MetaApplicationId?: string | null;
  MetaUiResourceId?: string | null;
};

/**
 * Role row slice that carries AccessUiResourceIds for UI projection sync.
 */
export type RoleAccessUiIdsRow = {
  Id?: unknown;
  AccessUiResourceIds?: string[];
};
