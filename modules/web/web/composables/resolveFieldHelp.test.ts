// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTermReference } from '@/core/service/i18n';
import { resolveFieldHelp } from './resolveFieldHelp';

const helpText = createTermReference('base', 'Short unique code used in references', {
  scope: 'base.model.Company.fields',
});

test('resolveFieldHelp: returns empty when help is absent', () => {
  expect(resolveFieldHelp({ meta: { string: 'Code' } as any })).toBe('');
  expect(resolveFieldHelp({ meta: null })).toBe('');
  expect(resolveFieldHelp({})).toBe('');
});

test('resolveFieldHelp: prefers FieldsGet translated help over msgid', () => {
  expect(
    resolveFieldHelp({
      meta: { help: 'Short unique code used in references', helpText },
      fieldsGetTranslatedHelp: '用于引用的短唯一编码',
    })
  ).toBe('用于引用的短唯一编码');
});

test('resolveFieldHelp: keeps FieldsGet help when meta has no helpText (not a msgid echo)', () => {
  expect(
    resolveFieldHelp({
      meta: { help: 'Short unique code used in references' },
      fieldsGetTranslatedHelp: '用于引用的短唯一编码',
    })
  ).toBe('用于引用的短唯一编码');
});

test('resolveFieldHelp: ignores FieldsGet msgid echo and falls back to help msgid without composer', () => {
  expect(
    resolveFieldHelp({
      meta: { help: 'Short unique code used in references', helpText },
      fieldsGetTranslatedHelp: 'Short unique code used in references',
    })
  ).toBe('Short unique code used in references');
});

test('resolveFieldHelp: ignores blank FieldsGet overlay values', () => {
  expect(
    resolveFieldHelp({
      meta: { help: 'Units per base currency' },
      fieldsGetTranslatedHelp: '   ',
    })
  ).toBe('Units per base currency');
});

test('resolveFieldHelp: returns plain help msgid', () => {
  expect(resolveFieldHelp({ meta: { help: 'Units per base currency' } })).toBe('Units per base currency');
});

test('resolveFieldHelp: uses helpText.src as msgid echo baseline when help msgid is blank', () => {
  expect(
    resolveFieldHelp({
      meta: { helpText },
      fieldsGetTranslatedHelp: 'Short unique code used in references',
    })
  ).toBe('Short unique code used in references');
});

test('resolveFieldHelp: translates via composer when helpText is present', () => {
  const composer = {
    t: (_key: string, src?: string) =>
      src === 'Short unique code used in references' ? '用于引用的短唯一编码' : String(src || _key),
  };
  expect(
    resolveFieldHelp({
      meta: { help: 'Short unique code used in references', helpText },
      composer,
    })
  ).toBe('用于引用的短唯一编码');
});

test('resolveFieldHelp: falls back through metaMsgid when helpText.src is non-string', () => {
  expect(
    resolveFieldHelp({
      meta: { help: 'Plain help', helpText: { ...helpText, src: undefined as any } },
      fieldsGetTranslatedHelp: 'overlay',
    })
  ).toBe('overlay');
});

test('resolveFieldHelp: ignores non-string FieldsGet overlay values', () => {
  expect(
    resolveFieldHelp({
      meta: { help: 'Units per base currency' },
      fieldsGetTranslatedHelp: null,
    })
  ).toBe('Units per base currency');
  expect(
    resolveFieldHelp({
      meta: { help: 'Units per base currency' },
      fieldsGetTranslatedHelp: 12 as any,
    })
  ).toBe('Units per base currency');
});

test('resolveFieldHelp: uses plain help when helpText is present but FieldsGet echo is skipped without msgid', () => {
  expect(
    resolveFieldHelp({
      meta: { helpText, help: undefined },
      fieldsGetTranslatedHelp: 'not-the-msgid',
    })
  ).toBe('not-the-msgid');
});

test('resolveFieldHelp: metaMsgid falls back to empty when help is not a string', () => {
  expect(
    resolveFieldHelp({
      meta: { help: 42 as any },
      fieldsGetTranslatedHelp: 'overlay-only',
    })
  ).toBe('overlay-only');
});

test('resolveFieldHelp: trims empty translateTerm results for helpText', () => {
  const emptySrc = createTermReference('base', '', { scope: 'base.model.Company.fields' });
  expect(
    resolveFieldHelp({
      meta: { helpText: emptySrc, help: '' },
      composer: { t: () => '' },
    })
  ).toBe('');
});

