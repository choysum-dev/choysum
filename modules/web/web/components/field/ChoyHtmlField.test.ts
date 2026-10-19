// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref } from 'vue';
import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyHtmlField from './ChoyHtmlField.vue';

function makeBinding(value: string | null, isEditMode = false): UseField {
  const current = ref(value);
  return {
    env: {
      isForm: true,
      fieldPresentation: 'form',
      isEditMode,
      viewMode: isEditMode ? 'edit' : 'display',
      fieldPrefix: null,
    },
    prop: 'Body',
    meta: { string: 'Body', type: 'html' },
    fieldRef: () => current,
    fieldRefOf: () => current,
    recordRef: () => computed(() => ({ Id: '1' })),
    registerFields: () => {},
    store: {},
    asView: () => ({ fieldValue: () => current }),
  } as unknown as UseField;
}

describe('ChoyHtmlField', () => {
  test('display mode renders sanitized html from the store binding', async () => {
    const m = mountApp(ChoyHtmlField as any, {
      props: {
        binding: makeBinding('<p>Hello</p><script>alert(1)</script>'),
        label: 'Body',
        renderMode: 'form',
      },
    });
    await flushPromises();
    const root = m.q('[data-anchor="choy.field-base"]');
    expect(root).toBeTruthy();
    expect(m.q('[data-slot="form-item"]')).toBeFalsy();
    const prose = m.q('.prose-choy');
    expect(prose?.innerHTML || '').toContain('Hello');
    expect(prose?.innerHTML || '').not.toContain('<script');
    m.unmount();
  });

  test('plaintext display strips tags', async () => {
    const m = mountApp(ChoyHtmlField as any, {
      props: {
        binding: makeBinding('<p>Plain <b>text</b></p>'),
        plaintext: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('.choy-field-display-text')?.textContent || '').toContain('Plain');
    expect(m.q('.choy-field-display-text')?.textContent || '').not.toContain('<b>');
    m.unmount();
  });
});
