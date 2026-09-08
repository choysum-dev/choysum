// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';
import { defineAction, getResourceDeclarationFromMeta } from '@/core/web/resource';
import { partnerRoutes } from '../route/routes';

const { _lt } = createTranslate('partner', { scope: 'web/views' });

test('partner resource wiring: each route declares expected actions', () => {
  const byName = Object.fromEntries(partnerRoutes.map(route => [String(route.name), route]));

  expect(getResourceDeclarationFromMeta(byName.PartnerList?.meta as any)?.actions).toEqual([
    'partner.action.partner_create',
    'partner.action.partner_edit',
    'partner.action.partner_delete',
    'partner.action.partner_copy',
    'partner.action.partner_open_detail',
  ]);
  expect(getResourceDeclarationFromMeta(byName.PartnerDetail?.meta as any)?.actions).toEqual([
    'partner.action.partner_create',
    'partner.action.partner_edit',
    'partner.action.partner_delete',
    'partner.action.partner_copy',
  ]);
  expect(getResourceDeclarationFromMeta(byName.PartnerCreate?.meta as any)?.actions).toEqual([
    'partner.action.partner_create',
    'partner.action.partner_edit',
    'partner.action.partner_delete',
    'partner.action.partner_copy',
  ]);
});

test('partner resource wiring: defineAction returns the registered action id', () => {
  expect(defineAction('partner.action.partner_open_detail', { title: _lt('Open Partner Detail') })).toBe(
    'partner.action.partner_open_detail'
  );
});
