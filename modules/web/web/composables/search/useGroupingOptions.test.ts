// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';

import { createTermReference } from '@/core/service/i18n';
import { useGroupingOptions } from './useGroupingOptions';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    getGlobalComposer: () => ({
      t: (_key: string, fallback: string) => (fallback === 'Status' ? '状态' : fallback),
    }),
  };
});

describe('useGroupingOptions field labels (T4.1)', () => {
  it('resolves labels via resolveFieldLabel (stringText), not bare prop names', () => {
    const statusText = createTermReference('demo', 'Status', {
      scope: 'demo.model.Widget.fields',
    });
    const store = {
      fieldsMetadata: {
        Status: { id: '1', type: 'selection', string: 'Status', stringText: statusText },
        Code: { id: '2', type: 'varchar', string: 'Code' },
        DeletedAt: { id: '9', type: 'datetime' },
      },
      getFieldsGetTranslatedString: () => undefined,
    } as any;

    const { availableGroupFields, groupTreeData } = useGroupingOptions(store);
    const status = availableGroupFields.value.find(f => f.prop === 'Status');
    const code = availableGroupFields.value.find(f => f.prop === 'Code');

    expect(status?.label).toBe('状态');
    expect(code?.label).toBe('Code');
    expect(availableGroupFields.value.some(f => f.prop === 'DeletedAt')).toBe(false);

    const statusNode = groupTreeData.value.find(n => n.id === 'f:Status');
    expect(statusNode?.label).toBe('状态');
  });

  it('prefers warm FieldsGet translated string when present', () => {
    const store = {
      fieldsMetadata: {
        Status: { id: '1', type: 'selection', string: 'Status' },
      },
      getFieldsGetTranslatedString: (name: string) => (name === 'Status' ? '启用状态' : undefined),
    } as any;

    const { availableGroupFields } = useGroupingOptions(store);
    expect(availableGroupFields.value[0]?.label).toBe('启用状态');
  });
});
