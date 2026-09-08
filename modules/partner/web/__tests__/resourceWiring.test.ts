// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getResourceDeclarationFromMeta } from '@/core/web/resource';
import { partnerRoutes } from '../route/routes';
import { partnerActions, partnerOpenDetailAction } from '../views/partner_actions';

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

test('partner resource wiring: shared partner_actions matches list/form declarations', () => {
  expect(partnerOpenDetailAction).toBe('partner.action.partner_open_detail');
  expect(partnerActions.create).toBe('partner.action.partner_create');
  expect(partnerActions.edit).toBe('partner.action.partner_edit');
  expect(partnerActions.copy).toBe('partner.action.partner_copy');
  expect(partnerActions.delete).toBe('partner.action.partner_delete');
});
