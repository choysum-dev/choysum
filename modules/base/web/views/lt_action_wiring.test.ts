// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineModelActions } from '@/core/web/resource';
import { createTermReference } from '@/core/service/i18n';

test('base view action wiring: defineModelActions yields ids for Company', () => {
  const actions = defineModelActions('base.Company', {
    entityTitle: createTermReference('base', 'Company', { scope: 'test' }),
  });
  expect(actions.create).toBeTruthy();
  expect(actions.delete).toBeTruthy();
  expect(actions.edit).toBeTruthy();
});

test('base view action wiring: defineModelActions yields ids for Language', () => {
  const actions = defineModelActions('base.Language', {
    entityTitle: createTermReference('base', 'Language', { scope: 'test' }),
  });
  expect(actions.create).toBeTruthy();
  expect(actions.delete).toBeTruthy();
  expect(actions.edit).toBeTruthy();
});

test('base view action wiring: defineModelActions yields ids for Currency', () => {
  const actions = defineModelActions('base.Currency', {
    entityTitle: createTermReference('base', 'Currency', { scope: 'test' }),
  });
  expect(actions.create).toBeTruthy();
  expect(actions.delete).toBeTruthy();
  expect(actions.edit).toBeTruthy();
});

test('base view action wiring: defineModelActions yields ids for UoM', () => {
  const actions = defineModelActions('base.UoM', {
    entityTitle: createTermReference('base', 'Unit of Measure', { scope: 'test' }),
  });
  expect(actions.create).toBeTruthy();
  expect(actions.delete).toBeTruthy();
  expect(actions.edit).toBeTruthy();
});
