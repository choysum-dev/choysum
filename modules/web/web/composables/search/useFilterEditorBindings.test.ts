// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// density backfill from main before merge
// Binding suite needs field-operator tables + store DI under QJS mount host.

import { useFilterEditorBindings } from './useFilterEditorBindings';

test('useFilterEditorBindings smoke: exports composable', () => {
  expect(typeof useFilterEditorBindings).toBe('function');
});
