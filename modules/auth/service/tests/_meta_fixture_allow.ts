// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Entry allowlist entries for reading meta IR tables from auth unit fixtures.
 * After removing repository app-wide meta RR bypass, fixture Searches must either
 * run under withPermissionGraphBypass or list these models on the entry allowlist.
 *
 * Prefer this read-only list. Tests that Create/Write/Delete MetaUiResource* rows
 * should also spread META_FIXTURE_RECORD_RULE_ALLOW_UI_MUTATE.
 */
export const META_FIXTURE_RECORD_RULE_ALLOW: string[] = [
  'meta.MetaApplication:read',
  'MetaApplication:read',
  'meta.MetaModel:read',
  'MetaModel:read',
  'meta.MetaField:read',
  'MetaField:read',
  'meta.MetaService:read',
  'MetaService:read',
  'meta.MetaModelData:read',
  'MetaModelData:read',
  'meta.MetaUiResource:read',
  'MetaUiResource:read',
  'meta.MetaUiResourceMenuRoute:read',
  'MetaUiResourceMenuRoute:read',
  'meta.MetaUiResourceRouteAction:read',
  'MetaUiResourceRouteAction:read',
];

/** Extra allowlist entries for fixtures that mutate MetaUiResource* rows. */
export const META_FIXTURE_RECORD_RULE_ALLOW_UI_MUTATE: string[] = [
  'meta.MetaUiResource:write',
  'meta.MetaUiResource:create',
  'meta.MetaUiResource:delete',
  'MetaUiResource:write',
  'MetaUiResource:create',
  'MetaUiResource:delete',
  'meta.MetaUiResourceMenuRoute:write',
  'meta.MetaUiResourceMenuRoute:create',
  'meta.MetaUiResourceMenuRoute:delete',
  'MetaUiResourceMenuRoute:write',
  'MetaUiResourceMenuRoute:create',
  'MetaUiResourceMenuRoute:delete',
  'meta.MetaUiResourceRouteAction:write',
  'meta.MetaUiResourceRouteAction:create',
  'meta.MetaUiResourceRouteAction:delete',
  'MetaUiResourceRouteAction:write',
  'MetaUiResourceRouteAction:create',
  'MetaUiResourceRouteAction:delete',
];
