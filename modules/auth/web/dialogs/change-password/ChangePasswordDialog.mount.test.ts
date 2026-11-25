// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { nextTick, reactive } from 'vue';

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChangePasswordDialog from './ChangePasswordDialog.vue';

describe('ChangePasswordDialog form-dialog shell', () => {
  test('mounts with shell chrome and onFooterCancel closes', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const props = reactive({ modelValue: true });
    const m = mountApp(ChangePasswordDialog as any, {
      props,
      reactiveProps: true,
      plugins: [pinia],
      on: {
        'onUpdate:modelValue': (v: boolean) => {
          props.modelValue = v;
        },
      },
    });
    await flushPromises();
    await nextTick();

    const ss = m.setupState() as {
      formViewRef: unknown;
      onFooterCancel: () => Promise<void>;
      dismissible: boolean | { value: boolean };
      embeddedFormChrome: Record<string, unknown>;
    };
    expect(ss.formViewRef != null || ss.embeddedFormChrome?.embedded === true).toBe(true);
    expect(ss.embeddedFormChrome?.embedded).toBe(true);
    expect(typeof ss.onFooterCancel).toBe('function');

    await ss.onFooterCancel();
    await flushPromises();
    expect(props.modelValue).toBe(false);
    m.unmount();
  });
});
