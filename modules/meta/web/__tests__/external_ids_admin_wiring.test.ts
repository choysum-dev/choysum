// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { metaSettingsAppMenu } from '../menu/menus';
import { metaRoutes } from '../route/routes';
import { getRouteActionsFromMeta } from '@/core/web/resource';
import { canShowDeclaredAction } from '@/web/web/components/view/actionVisibility';

/**
 * Nail Owner=meta External IDs ledger invariants:
 * Settings → meta only; sys.admin; list + detail; no /new; no per-app host.
 */
test('Settings → meta mounts External IDs for sys.admin only', () => {
  const ids = metaSettingsAppMenu.children?.map(c => c.id) ?? [];
  expect(ids).toContain('meta.menu.external_ids');

  const item = metaSettingsAppMenu.children?.find(c => c.id === 'meta.menu.external_ids') as any;
  expect(item?.path).toBe('/web/external-ids');
  expect(item?.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
});

test('External IDs routes are Owner-meta list+detail only (no /new, no :application)', () => {
  const list = metaRoutes.find(r => r.name === 'MetaModelDataList') as any;
  const detail = metaRoutes.find(r => r.name === 'MetaModelDataDetail') as any;
  expect(list).toBeTruthy();
  expect(detail).toBeTruthy();

  expect(list.path).toBe('web/external-ids');
  expect(detail.path).toBe('web/external-ids/:id');
  expect(list.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
  expect(detail.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
  expect(getRouteActionsFromMeta(list.meta)).toEqual([]);
  expect(getRouteActionsFromMeta(detail.meta)).toEqual([]);

  for (const r of [list, detail]) {
    expect(String(r.path)).not.toContain(':application');
    expect(String(r.path)).not.toContain('/new');
  }

  expect(metaRoutes.some(r => String(r.path).includes('external-ids/new'))).toBe(false);
});

test('External IDs Form omits mutating action-ids so Edit/Create/Copy/Delete stay hidden', () => {
  // ChoyFormView uses canShowDeclaredAction for mutating slots (empty id → false).
  expect(canShowDeclaredAction(undefined, () => true)).toBe(false);
  expect(canShowDeclaredAction('', () => true)).toBe(false);
});
