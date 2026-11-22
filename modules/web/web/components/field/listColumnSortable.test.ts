// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  defaultListColumnSortable,
  defaultListColumnSortableForFieldType,
  isListColumnOrderableByStorage,
} from './listColumnSortable';

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

test('isListColumnOrderableByStorage gates non-column / non-sql fields', () => {
  expect(isListColumnOrderableByStorage(undefined)).toBe(true);
  expect(isListColumnOrderableByStorage({ shouldCreateColumn: true })).toBe(true);
  expect(isListColumnOrderableByStorage({ computedKind: 'sql' })).toBe(true);
  expect(isListColumnOrderableByStorage({ shouldCreateColumn: false })).toBe(false);
  expect(isListColumnOrderableByStorage({ shouldCreateColumn: false, computedKind: 'sql' })).toBe(true);
  expect(isListColumnOrderableByStorage({ storageKind: 'virtual' })).toBe(false);
  expect(isListColumnOrderableByStorage({ storageKind: 'virtualSql' })).toBe(false);
  expect(isListColumnOrderableByStorage({ storageKind: 'column' })).toBe(true);
});

test('defaultListColumnSortable combines type and storage gates', () => {
  expect(defaultListColumnSortable('decimal', { shouldCreateColumn: true })).toBe(true);
  expect(defaultListColumnSortable('decimal', { shouldCreateColumn: false })).toBe(false);
  expect(defaultListColumnSortable('ManyToMany', { shouldCreateColumn: true })).toBe(false);
});
