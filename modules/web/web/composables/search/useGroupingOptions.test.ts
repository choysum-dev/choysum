// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTermReference } from '@/core/service/i18n';
import type { ComposerLike } from '@/web/web/i18n';
import { useGroupingOptions } from './useGroupingOptions';

function installComposer(composer: ComposerLike) {
  (globalThis as { $i18n?: ComposerLike }).$i18n = composer;
}

function clearComposer() {
  delete (globalThis as { $i18n?: ComposerLike }).$i18n;
  const win = (globalThis as { window?: { $i18n?: ComposerLike } }).window;
  if (win) delete win.$i18n;
}

afterEach(() => {
  clearComposer();
});

describe('useGroupingOptions field labels', () => {
  test('resolves labels via resolveFieldLabel (stringText), not bare prop names', () => {
    installComposer({
      t: (_key: string, fallback: string) => (fallback === 'Status' ? '状态' : fallback),
    });
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

  test('prefers warm FieldsGet translated string when present', () => {
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
