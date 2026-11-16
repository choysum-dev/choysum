// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { bcp47FromCode } from './bcp47';

test('bcp47FromCode: underscore to hyphen', () => {
  expect(bcp47FromCode('zh_CN')).toBe('zh-CN');
  expect(bcp47FromCode('en_US')).toBe('en-US');
  expect(bcp47FromCode('pt_BR')).toBe('pt-BR');
});

test('bcp47FromCode: empty falls back to en-US', () => {
  expect(bcp47FromCode('')).toBe('en-US');
  expect(bcp47FromCode('   ')).toBe('en-US');
});
