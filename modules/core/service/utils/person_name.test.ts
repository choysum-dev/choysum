// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  DEFAULT_PERSON_NAME_FORMAT,
  formatPersonName,
  isValidPersonNameFormat,
  normalizePersonNameFormat,
} from './person_name';

test('normalizePersonNameFormat: empty/invalid → default', () => {
  expect(normalizePersonNameFormat('')).toBe(DEFAULT_PERSON_NAME_FORMAT);
  expect(normalizePersonNameFormat('   ')).toBe(DEFAULT_PERSON_NAME_FORMAT);
  expect(normalizePersonNameFormat('Hello')).toBe(DEFAULT_PERSON_NAME_FORMAT);
  expect(normalizePersonNameFormat('{MiddleName}')).toBe(DEFAULT_PERSON_NAME_FORMAT);
  expect(normalizePersonNameFormat('{FirstName} {LastName}')).toBe('{FirstName} {LastName}');
  expect(normalizePersonNameFormat('  {LastName}{FirstName}  ')).toBe('{LastName}{FirstName}');
});

test('isValidPersonNameFormat: requires FirstName or LastName token', () => {
  expect(isValidPersonNameFormat('{FirstName}')).toBe(true);
  expect(isValidPersonNameFormat('{LastName}')).toBe(true);
  expect(isValidPersonNameFormat('{FirstName}{LastName}')).toBe(true);
  expect(isValidPersonNameFormat('plain')).toBe(false);
  expect(isValidPersonNameFormat('{Foo}')).toBe(false);
});

test('formatPersonName: first_last and last_first templates', () => {
  expect(formatPersonName({ FirstName: '步科', LastName: '王' }, '{FirstName} {LastName}')).toBe('步科 王');
  expect(formatPersonName({ FirstName: '步科', LastName: '王' }, '{LastName}{FirstName}')).toBe('王步科');
});

test('formatPersonName: empty parts collapse whitespace', () => {
  expect(formatPersonName({ FirstName: 'Ada', LastName: '' }, '{FirstName} {LastName}')).toBe('Ada');
  expect(formatPersonName({ FirstName: '', LastName: 'Lovelace' }, '{FirstName} {LastName}')).toBe('Lovelace');
  expect(formatPersonName({ FirstName: '', LastName: '' }, '{FirstName} {LastName}')).toBe('');
});

test('formatPersonName: strips unknown tokens', () => {
  expect(
    formatPersonName({ FirstName: 'A', LastName: 'B' }, '{FirstName} {MiddleName} {LastName}'),
  ).toBe('A B');
});

test('formatPersonName: invalid template falls back to default', () => {
  expect(formatPersonName({ FirstName: 'A', LastName: 'B' }, 'no-tokens')).toBe('A B');
});
