// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { createTermReference } from '@/core/service/i18n';
import { resolveFieldHelp } from './resolveFieldHelp';

describe('resolveFieldHelp', () => {
  const helpText = createTermReference('base', 'Short unique code used in references', {
    scope: 'base.model.Company.fields',
  });

  it('returns empty when help is absent', () => {
    expect(resolveFieldHelp({ meta: { string: 'Code' } as any })).toBe('');
  });

  it('prefers FieldsGet translated help over msgid', () => {
    expect(
      resolveFieldHelp({
        meta: { help: 'Short unique code used in references', helpText },
        fieldsGetTranslatedHelp: '用于引用的短唯一编码',
      })
    ).toBe('用于引用的短唯一编码');
  });

  it('ignores FieldsGet msgid echo and falls back to help msgid without composer', () => {
    expect(
      resolveFieldHelp({
        meta: { help: 'Short unique code used in references', helpText },
        fieldsGetTranslatedHelp: 'Short unique code used in references',
      })
    ).toBe('Short unique code used in references');
  });

  it('returns plain help msgid', () => {
    expect(resolveFieldHelp({ meta: { help: 'Units per base currency' } })).toBe('Units per base currency');
  });
});
