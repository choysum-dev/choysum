// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defaultListColumnSortableForFieldType } from './listColumnSortable';

test('defaultListColumnSortableForFieldType allows scalars and ManyToOne* ', () => {
  for (const t of [
    'varchar',
    'Char',
    'text',
    'int',
    'bigint',
    'decimal',
    'monetary',
    'boolean',
    'date',
    'datetime',
    'time',
    'selection',
    'ManyToOne',
    'ManyToOneRef',
  ]) {
    expect(defaultListColumnSortableForFieldType(t)).toBe(true);
  }
});

test('defaultListColumnSortableForFieldType rejects ToMany* and blob-like types', () => {
  for (const t of [
    'ManyToMany',
    'ManyToManyRef',
    'OneToMany',
    'image',
    'binary',
    'html',
    'json',
    'properties',
    'PropertiesDefinition',
  ]) {
    expect(defaultListColumnSortableForFieldType(t)).toBe(false);
  }
});

test('defaultListColumnSortableForFieldType is false for empty / unknown types', () => {
  expect(defaultListColumnSortableForFieldType('')).toBe(false);
  expect(defaultListColumnSortableForFieldType(undefined)).toBe(false);
  expect(defaultListColumnSortableForFieldType('unknown-widget')).toBe(false);
});
