// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTermReference } from '@/core/service/i18n';
import { resolveSelectionOptionLabel } from './resolveSelectionOptionLabel';

const fieldStringText = createTermReference('auth', 'Theme', {
  scope: 'auth.model.User.fields',
});

test('resolveSelectionOptionLabel: translates msgid echo via field stringText scope', () => {
  expect(
    resolveSelectionOptionLabel({
      label: 'Light',
      fieldStringText,
      composer: {
        t: (_key: string, fallback: string) => (fallback === 'Light' ? '浅色' : fallback),
      },
    }),
  ).toBe('浅色');
});

test('resolveSelectionOptionLabel: already-translated labels pass through on catalog miss', () => {
  expect(
    resolveSelectionOptionLabel({
      label: '浅色',
      fieldStringText,
      composer: {
        t: (_key: string, fallback: string) => fallback,
      },
    }),
  ).toBe('浅色');
});

test('resolveSelectionOptionLabel: without stringText or composer returns label', () => {
  expect(resolveSelectionOptionLabel({ label: 'Dark' })).toBe('Dark');
  expect(
    resolveSelectionOptionLabel({
      label: 'System',
      fieldStringText,
    }),
  ).toBe('System');
});

test('resolveSelectionOptionLabel: empty label stays empty', () => {
  expect(resolveSelectionOptionLabel({ label: '' })).toBe('');
  expect(resolveSelectionOptionLabel({ label: '   ' })).toBe('');
});
