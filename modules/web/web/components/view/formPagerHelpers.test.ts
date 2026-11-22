// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  formPagerCurrentInputCh,
  parseFormPagerCurrentDraft,
  resolvePagerInputEl,
} from './formPagerHelpers';

test('parseFormPagerCurrentDraft accepts positive integers and clamps to total', () => {
  expect(parseFormPagerCurrentDraft('152', 249)).toBe(152);
  expect(parseFormPagerCurrentDraft('999', 249)).toBe(249);
  expect(parseFormPagerCurrentDraft('1', 249)).toBe(1);
});

test('parseFormPagerCurrentDraft rejects empty or non-integer drafts', () => {
  expect(parseFormPagerCurrentDraft('', 10)).toBeNull();
  expect(parseFormPagerCurrentDraft('  ', 10)).toBeNull();
  expect(parseFormPagerCurrentDraft('1-2', 10)).toBeNull();
  expect(parseFormPagerCurrentDraft('abc', 10)).toBeNull();
  expect(parseFormPagerCurrentDraft('0', 10)).toBeNull();
  expect(parseFormPagerCurrentDraft('-3', 10)).toBeNull();
  expect(parseFormPagerCurrentDraft('5', 0)).toBeNull();
});

test('formPagerCurrentInputCh sizes from digit runs plus padding slack', () => {
  // total "249" → 3+3=6ch
  expect(formPagerCurrentInputCh({ currentLabel: '9', totalLabel: '249' })).toBe(6);
  expect(formPagerCurrentInputCh({ draft: '200', currentLabel: '9', totalLabel: '99' })).toBe(6);
  expect(formPagerCurrentInputCh({ currentLabel: '1', totalLabel: '9' })).toBe(5);
  expect(formPagerCurrentInputCh({ draft: '12345678901234', currentLabel: '1', totalLabel: '9' })).toBe(12);
});

test('resolvePagerInputEl accepts plain inputs and component $el wrappers', () => {
  expect(resolvePagerInputEl(null)).toBeNull();
  expect(resolvePagerInputEl(undefined)).toBeNull();
  const input = { focus() {}, select() {} } as HTMLInputElement;
  expect(resolvePagerInputEl(input)).toBe(input);
  // Component refs expose the native node on `$el`.
  const wrapped = document.createElement('input');
  expect(resolvePagerInputEl({ $el: wrapped })).toBe(wrapped);
  expect(resolvePagerInputEl({ $el: { tagName: 'DIV' } as any })).toBeNull();
  expect(resolvePagerInputEl({} as any)).toBeNull();
});
