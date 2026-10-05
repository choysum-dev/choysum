// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type MenuItem } from '@/core/web/menu';
import { defineMenu } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('base', { scope: 'web/menu/menus' });

export const baseMenus: MenuItem[] = [
  defineMenu('base.menu.root', {
    title: _lt('Master Data'),
    sequence: 20,
    children: [
      defineMenu('base.menu.company', { title: _lt('Company Management'), path: '/base/companies', sequence: 10 }),
      defineMenu('base.menu.address', { title: _lt('Address Management'), path: '/base/addresses', sequence: 20 }),
      defineMenu('base.menu.bank', { title: _lt('Bank Management'), path: '/base/banks', sequence: 30 }),
      defineMenu('base.menu.country', { title: _lt('Country Management'), path: '/base/countries', sequence: 40 }),
      defineMenu('base.menu.state', { title: _lt('State Management'), path: '/base/states', sequence: 50 }),
      defineMenu('base.menu.city', { title: _lt('City Management'), path: '/base/cities', sequence: 60 }),
      defineMenu('base.menu.currency', { title: _lt('Currency Management'), path: '/base/currencies', sequence: 70 }),
      defineMenu('base.menu.exchange_rate', { title: _lt('Exchange Rate Management'), path: '/base/exchange-rates', sequence: 80 }),
      defineMenu('base.menu.language', { title: _lt('Language Management'), path: '/base/languages', sequence: 90 }),
      defineMenu('base.menu.terminology', {
        title: _lt('Terminology Editor'),
        path: '/base/terminology',
        sequence: 95,
        // Role-gated only; see base.route.terminology_editor comment.
        defaultRoles: ['terminology.editor'],
      }),
      defineMenu('base.menu.sequence', { title: _lt('Sequence Management'), path: '/base/sequences', sequence: 110 }),
      defineMenu('base.menu.sequence_idempotency', { title: _lt('Sequence Idempotency Record'), path: '/base/sequence-idempotencies', sequence: 120 }),
      defineMenu('base.menu.uom_category', { title: _lt('Unit of Measure Category'), path: '/base/uom-categories', sequence: 130 }),
      defineMenu('base.menu.uom', { title: _lt('Unit of Measure'), path: '/base/uoms', sequence: 140 }),
    ],
  }),
];
