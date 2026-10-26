// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { componentHintFromScope } from './componentHint';

test('componentHintFromScope: strips @field suffix for page scopes', () => {
  expect(componentHintFromScope('web/pages/Login@title')).toBe('web/pages/Login');
  expect(
    componentHintFromScope('web/components/layout/ChoyAppHeader@brand'),
  ).toBe('web/components/layout/ChoyAppHeader');
  expect(componentHintFromScope('game.rescue')).toBe('game.rescue');
  expect(componentHintFromScope('')).toBe('');
});

