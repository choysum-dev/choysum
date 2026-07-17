// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type MenuItem } from '@/core/web/menu';
import { OfficeBuilding, School } from '@element-plus/icons-vue';
import { defineMenu } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _td } = createTranslate('base', { scope: 'web/menu/menus' });

export const baseMenus: MenuItem[] = [
  defineMenu('base.menu.root', {
    title: _td('Master Data'),
    icon: OfficeBuilding,
    sequence: 20,
    children: [
      defineMenu('base.menu.company', { title: _td('Company Management'), icon: School, path: '/base/companies', sequence: 10 }),
      defineMenu('base.menu.address', { title: _td('Address Management'), icon: School, path: '/base/addresses', sequence: 20 }),
      defineMenu('base.menu.bank', { title: _td('Bank Management'), icon: School, path: '/base/banks', sequence: 30 }),
      defineMenu('base.menu.country', { title: _td('Country Management'), icon: School, path: '/base/countries', sequence: 40 }),
      defineMenu('base.menu.state', { title: _td('State Management'), icon: School, path: '/base/states', sequence: 50 }),
      defineMenu('base.menu.city', { title: _td('City Management'), icon: School, path: '/base/cities', sequence: 60 }),
      defineMenu('base.menu.currency', { title: _td('Currency Management'), icon: School, path: '/base/currencies', sequence: 70 }),
      defineMenu('base.menu.exchange_rate', { title: _td('Exchange Rate Management'), icon: School, path: '/base/exchange-rates', sequence: 80 }),
      defineMenu('base.menu.language', { title: _td('Language Management'), icon: School, path: '/base/languages', sequence: 90 }),
      defineMenu('base.menu.terminology', {
        title: _td('Terminology Editor'),
        icon: School,
        path: '/base/terminology',
        sequence: 95,
        defaultRoles: ['terminology.editor'],
        requires: [{ model: 'web.I18n', method: 'SearchTerms' }],
      }),
      defineMenu('base.menu.locale', { title: _td('Locale Management'), icon: School, path: '/base/locales', sequence: 100 }),
      defineMenu('base.menu.sequence', { title: _td('Sequence Management'), icon: School, path: '/base/sequences', sequence: 110 }),
      defineMenu('base.menu.sequence_idempotency', { title: _td('Sequence Idempotency Record'), icon: School, path: '/base/sequence-idempotencies', sequence: 120 }),
      defineMenu('base.menu.uom_category', { title: _td('Unit of Measure Category'), icon: School, path: '/base/uom-categories', sequence: 130 }),
      defineMenu('base.menu.uom', { title: _td('Unit of Measure'), icon: School, path: '/base/uoms', sequence: 140 }),
    ],
  }),
];
