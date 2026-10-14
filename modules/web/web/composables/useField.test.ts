// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, provide } from 'vue';

import { mountApp } from '@/web/web/__tests__/mountApp';
import { useField } from '@/web/web/composables/useField';
import type { ViewKindId, ViewMode } from '@/web/web/components/view/ChoyViewScope.vue';
import { useProvideViewScope } from '@/web/web/components/view/viewScopeKeys';

function mountFieldEnv(opts: {
  viewContainer?: ViewKindId;
  viewMode?: ViewMode;
  formRoot?: { draft: Record<string, unknown> } | null;
}) {
  const Probe = defineComponent({
    setup() {
      const field = useField({ prop: 'Name' });
      return () => h('span', { 'data-edit': String(field.env.isEditMode) });
    },
  });

  const Host = defineComponent({
    setup() {
      useProvideViewScope({
        kind: opts.viewContainer ?? 'List',
        mode: opts.viewMode ?? 'display',
      });
      if (opts.formRoot) provide('form-root', opts.formRoot);
      return () => h(Probe);
    },
  });

  return mountApp(Host);
}

describe('useField FieldEnv.isEditMode (List + form-root, D4)', () => {
  test('is false for List display without form-root', () => {
    const { unmount, q } = mountFieldEnv({ viewContainer: 'List', viewMode: 'display', formRoot: null });
    expect(q('span')?.getAttribute('data-edit')).toBe('false');
    unmount();
  });

  test('is false for List edit mode without form-root', () => {
    const { unmount, q } = mountFieldEnv({ viewContainer: 'List', viewMode: 'edit', formRoot: null });
    expect(q('span')?.getAttribute('data-edit')).toBe('false');
    unmount();
  });

  test('is false for List edit mode with form-root but null draft', () => {
    const { unmount, q } = mountFieldEnv({
      viewContainer: 'List',
      viewMode: 'edit',
      formRoot: { draft: null as any },
    });
    expect(q('span')?.getAttribute('data-edit')).toBe('false');
    unmount();
  });

  test('is true for List edit mode with row form-root draft', () => {
    const { unmount, q } = mountFieldEnv({
      viewContainer: 'List',
      viewMode: 'edit',
      formRoot: { draft: { Id: '1', Name: 'Row' } },
    });
    expect(q('span')?.getAttribute('data-edit')).toBe('true');
    unmount();
  });

  test('is true for Form edit mode without form-root', () => {
    const { unmount, q } = mountFieldEnv({ viewContainer: 'Form', viewMode: 'edit', formRoot: null });
    expect(q('span')?.getAttribute('data-edit')).toBe('true');
    unmount();
  });
});
