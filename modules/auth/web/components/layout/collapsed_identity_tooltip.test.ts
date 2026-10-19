// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { collapsedIdentityTooltip } from './collapsed_identity_tooltip';

test('collapsedIdentityTooltip: returns identity label when the menu is closed', () => {
  expect(collapsedIdentityTooltip(false, '主公司')).toBe('主公司');
  expect(collapsedIdentityTooltip(false, 'Admin')).toBe('Admin');
});

test('collapsedIdentityTooltip: omits tip while the menu is open', () => {
  expect(collapsedIdentityTooltip(true, '主公司')).toBeUndefined();
  expect(collapsedIdentityTooltip(true, 'Admin')).toBeUndefined();
});

test('collapsedIdentityTooltip: omits empty or whitespace labels', () => {
  expect(collapsedIdentityTooltip(false, '')).toBeUndefined();
  expect(collapsedIdentityTooltip(false, '   ')).toBeUndefined();
});
