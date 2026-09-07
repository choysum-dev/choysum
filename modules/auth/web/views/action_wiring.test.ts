// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineModelActions } from '@/core/web/resource';
import { createTermReference } from '@/core/service/i18n';

test('auth view action wiring: defineModelActions yields ids for User', () => {
  const actions = defineModelActions('auth.User', { entityTitle: createTermReference('auth', 'User', { scope: 'test' }) });
  expect(actions.create).toBeTruthy();
  expect(actions.delete).toBeTruthy();
  expect(actions.edit).toBeTruthy();
});

test('auth view action wiring: defineModelActions yields ids for Role', () => {
  const actions = defineModelActions('auth.Role', { entityTitle: createTermReference('auth', 'Role', { scope: 'test' }) });
  expect(actions.create).toBeTruthy();
  expect(actions.delete).toBeTruthy();
});

test('auth view action wiring: defineModelActions yields ids for Session', () => {
  const actions = defineModelActions('auth.Session', { entityTitle: createTermReference('auth', 'Session', { scope: 'test' }) });
  expect(actions.create).toBeTruthy();
  expect(actions.delete).toBeTruthy();
});

test('auth view action wiring: defineModelActions yields ids for Token', () => {
  const actions = defineModelActions('auth.Token', { entityTitle: createTermReference('auth', 'Token', { scope: 'test' }) });
  expect(actions.create).toBeTruthy();
  expect(actions.delete).toBeTruthy();
});
