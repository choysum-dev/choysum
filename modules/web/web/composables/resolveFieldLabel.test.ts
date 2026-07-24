// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { createTermReference } from '@/core/service/i18n';
import { resolveFieldLabel } from './resolveFieldLabel';

describe('resolveFieldLabel (D6)', () => {
  const stringText = createTermReference('auth', 'Access Token ID', {
    scope: 'auth.model.Session.fields',
  });

  it('uses explicit props.label including empty string', () => {
    expect(
      resolveFieldLabel({
        label: 'Custom',
        prop: 'AccessTokenId',
        meta: { string: 'Access Token ID', stringText },
      })
    ).toBe('Custom');
    expect(
      resolveFieldLabel({
        label: '',
        prop: 'AccessTokenId',
        meta: { string: 'Access Token ID', stringText },
      })
    ).toBe('');
  });

  it('prefers FieldsGet translated string over stringText', () => {
    expect(
      resolveFieldLabel({
        prop: 'AccessTokenId',
        meta: { string: 'Access Token ID', stringText },
        fieldsGetTranslatedString: '访问令牌 ID',
        composer: { t: () => 'from-composer' },
      })
    ).toBe('访问令牌 ID');
  });

  it('ignores FieldsGet msgid echo and translates via stringText (selection ensure path)', () => {
    expect(
      resolveFieldLabel({
        prop: 'Timezone',
        meta: { string: 'Time Zone', stringText: createTermReference('base', 'Time Zone', { scope: 'base.model.Company.fields' }) },
        fieldsGetTranslatedString: 'Time Zone',
        composer: {
          t: (_key: string, fallback: string) => (fallback === 'Time Zone' ? '时区' : fallback),
        },
      })
    ).toBe('时区');
  });

  it('translates via stringText when composer is available', () => {
    expect(
      resolveFieldLabel({
        prop: 'AccessTokenId',
        meta: { string: 'Access Token ID', stringText },
        composer: {
          t: (key: string, fallback: string) => (key === stringText.key ? '访问令牌 ID' : fallback),
        },
      })
    ).toBe('访问令牌 ID');
  });

  it('falls back to meta.string msgid without composer or stringText', () => {
    expect(
      resolveFieldLabel({
        prop: 'AccessTokenId',
        meta: { string: 'Access Token ID' },
      })
    ).toBe('Access Token ID');
  });

  it('falls back to leaf prop name when meta is missing', () => {
    expect(resolveFieldLabel({ prop: 'Partner.Name' })).toBe('Name');
    expect(resolveFieldLabel({ prop: 'Status' })).toBe('Status');
  });

  it('does not throw without FieldsGet or vocabulary', () => {
    const statusText = createTermReference('auth', 'Status', {
      scope: 'auth.model.Session.fields',
    });
    expect(() =>
      resolveFieldLabel({
        prop: 'Status',
        meta: { string: 'Status', stringText: statusText },
      })
    ).not.toThrow();
    expect(
      resolveFieldLabel({
        prop: 'Status',
        meta: { string: 'Status', stringText: statusText },
      })
    ).toBe('Status');
  });
});
