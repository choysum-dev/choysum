// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Entry allowlist entries for reading/writing meta IR tables from auth unit fixtures.
 * After removing repository app-wide meta RR bypass, fixture Searches must either
 * run under withPermissionGraphBypass or list these models on the entry allowlist.
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
  'meta.MetaUiResource:write',
  'meta.MetaUiResource:create',
  'meta.MetaUiResource:delete',
  'MetaUiResource:read',
  'MetaUiResource:write',
  'MetaUiResource:create',
  'MetaUiResource:delete',
  'meta.MetaUiResourceMenuRoute:read',
  'meta.MetaUiResourceMenuRoute:write',
  'meta.MetaUiResourceMenuRoute:create',
  'meta.MetaUiResourceMenuRoute:delete',
  'MetaUiResourceMenuRoute:read',
  'MetaUiResourceMenuRoute:write',
  'MetaUiResourceMenuRoute:create',
  'MetaUiResourceMenuRoute:delete',
  'meta.MetaUiResourceRouteAction:read',
  'meta.MetaUiResourceRouteAction:write',
  'meta.MetaUiResourceRouteAction:create',
  'meta.MetaUiResourceRouteAction:delete',
  'MetaUiResourceRouteAction:read',
  'MetaUiResourceRouteAction:write',
  'MetaUiResourceRouteAction:create',
  'MetaUiResourceRouteAction:delete',
];
