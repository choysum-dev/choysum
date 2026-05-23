// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type MenuItem } from '@/core/web/menu';
import { OfficeBuilding, School } from '@element-plus/icons-vue';
import { defineMenu } from '@/core/web/resource';

export const baseMenus: MenuItem[] = [
  defineMenu('base.menu.root', {
    title: 'Master Data',
    icon: OfficeBuilding,
    sequence: 20,
    children: [
      defineMenu('base.menu.company', { title: 'Company Management', icon: School, path: '/base/companies', sequence: 10 }),
      defineMenu('base.menu.address', { title: 'Address Management', icon: School, path: '/base/addresses', sequence: 20 }),
      defineMenu('base.menu.bank', { title: 'Bank Management', icon: School, path: '/base/banks', sequence: 30 }),
      defineMenu('base.menu.country', { title: 'Country Management', icon: School, path: '/base/countries', sequence: 40 }),
      defineMenu('base.menu.state', { title: 'State Management', icon: School, path: '/base/states', sequence: 50 }),
      defineMenu('base.menu.city', { title: 'City Management', icon: School, path: '/base/cities', sequence: 60 }),
      defineMenu('base.menu.currency', { title: 'Currency Management', icon: School, path: '/base/currencies', sequence: 70 }),
      defineMenu('base.menu.exchange_rate', { title: 'Exchange Rate Management', icon: School, path: '/base/exchange-rates', sequence: 80 }),
      defineMenu('base.menu.language', { title: 'Language Management', icon: School, path: '/base/languages', sequence: 90 }),
      defineMenu('base.menu.locale', { title: 'Locale Management', icon: School, path: '/base/locales', sequence: 100 }),
      defineMenu('base.menu.sequence', { title: 'Sequence Management', icon: School, path: '/base/sequences', sequence: 110 }),
      defineMenu('base.menu.sequence_idempotency', { title: 'Sequence Idempotency Record', icon: School, path: '/base/sequence-idempotencies', sequence: 120 }),
      defineMenu('base.menu.uom_category', { title: 'Unit of Measure Category', icon: School, path: '/base/uom-categories', sequence: 130 }),
      defineMenu('base.menu.uom', { title: 'Unit of Measure', icon: School, path: '/base/uoms', sequence: 140 }),
    ],
  }),
];
