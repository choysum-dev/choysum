// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Authz Search / grant row shapes shared across auth service helpers.
 *
 * Kept in `auth` (not `core`): these are meta/auth domain projections, not ORM
 * primitives. Meta row aliases use Partial&lt;Pick&lt;Model&gt;&gt; so createServiceByModel
 * Search results assign without hand-rolled bags.
 *
 * Bags use PascalCase field names only (Id-only refs).
 */

import type MetaUiResource from '@/meta/service/models/ui_resource';
import type MetaUiResourceMenuRoute from '@/meta/service/models/ui_resource_menu_route';
import type MetaUiResourceRouteAction from '@/meta/service/models/ui_resource_route_action';
import type RoleUiResource from './role_ui_resource';

/**
 * `meta.MetaUiResource.Search` projection used by method-access and permission-state UI.
 * Field set is the union of those call sites (extra keys stay optional).
 */
export type MetaUiResourceAuthzRow = Partial<
  Pick<MetaUiResource, 'Id' | 'Name' | 'Type' | 'ParentId' | 'MetaApplicationId' | 'Requires'>
>;

/**
 * `meta.MetaUiResourceMenuRoute.Search` projection for menu↔route edges.
 */
export type MetaUiResourceMenuRouteRow = Partial<Pick<MetaUiResourceMenuRoute, 'MenuUiResourceId' | 'RouteUiResourceId'>>;

/**
 * `meta.MetaUiResourceRouteAction.Search` projection for route↔action edges.
 */
export type MetaUiResourceRouteActionRow = Partial<Pick<MetaUiResourceRouteAction, 'RouteUiResourceId' | 'ActionUiResourceId'>>;

/** `auth.RoleUiResource` grant row used by UI sync (Id optional on synthetic allow rows). */
export type RoleUiResourceGrantRow = Partial<Pick<RoleUiResource, 'Id' | 'Mode' | 'MetaApplicationId' | 'MetaUiResourceId'>>;

/**
 * Role rows passed into AccessUiResourceIds hydration.
 * Kept structural: callers pass generic `RowOrProjected` which is not assignable to `Pick<Role, …>`.
 */
export type RoleAccessUiIdsRow = {
  Id?: unknown;
  AccessUiResourceIds?: string[];
};
