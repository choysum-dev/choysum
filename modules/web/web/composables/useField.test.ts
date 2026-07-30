// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, provide, ref } from 'vue';
import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { useField } from '@/web/web/composables/useField';
import type { ViewContainer, ViewMode } from '@/web/web/components/view/OViewScope.vue';

function mountFieldEnv(opts: {
  viewContainer?: ViewContainer;
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
      provide('view-container', ref(opts.viewContainer ?? 'List'));
      provide('view-mode', ref(opts.viewMode ?? 'display'));
      if (opts.formRoot) provide('form-root', opts.formRoot);
      return () => h(Probe);
    },
  });

  return mount(Host);
}

describe('useField FieldEnv.isEditMode (List + form-root, D4)', () => {
  it('is false for List display without form-root', () => {
    const wrapper = mountFieldEnv({ viewContainer: 'List', viewMode: 'display', formRoot: null });
    expect(wrapper.get('span').attributes('data-edit')).toBe('false');
  });

  it('is false for List edit mode without form-root', () => {
    const wrapper = mountFieldEnv({ viewContainer: 'List', viewMode: 'edit', formRoot: null });
    expect(wrapper.get('span').attributes('data-edit')).toBe('false');
  });

  it('is false for List edit mode with form-root but null draft', () => {
    const wrapper = mountFieldEnv({
      viewContainer: 'List',
      viewMode: 'edit',
      formRoot: { draft: null as any },
    });
    expect(wrapper.get('span').attributes('data-edit')).toBe('false');
  });

  it('is true for List edit mode with row form-root draft', () => {
    const wrapper = mountFieldEnv({
      viewContainer: 'List',
      viewMode: 'edit',
      formRoot: { draft: { Id: '1', Name: 'Row' } },
    });
    expect(wrapper.get('span').attributes('data-edit')).toBe('true');
  });

  it('is true for Form edit mode without form-root', () => {
    const wrapper = mountFieldEnv({ viewContainer: 'Form', viewMode: 'edit', formRoot: null });
    expect(wrapper.get('span').attributes('data-edit')).toBe('true');
  });
});
